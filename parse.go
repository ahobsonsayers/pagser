package pagser

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cast"
)

// Parse parse html to struct
func (p *Pagser) Parse(v interface{}, document string) (err error) {
	reader, err := goquery.NewDocumentFromReader(strings.NewReader(document))
	if err != nil {
		return err
	}
	return p.ParseDocument(v, reader)
}

// ParseReader parse html to struct
func (p *Pagser) ParseReader(v interface{}, reader io.Reader) (err error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}
	return p.ParseDocument(v, doc)
}

// ParseDocument parse document to struct
func (p *Pagser) ParseDocument(v interface{}, document *goquery.Document) (err error) {
	return p.ParseSelection(v, document.Selection)
}

// ParseSelection parse selection to struct
func (p *Pagser) ParseSelection(v interface{}, selection *goquery.Selection) (err error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("%v is non-pointer", val.Type())
	}

	if val.IsNil() {
		return fmt.Errorf("%v is nil", val.Type())
	}

	elem := val.Elem()

	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("%v is not a struct", elem.Type())
	}

	return p.doParse(val, nil, selection)
}

// ParseSelection parse selection to struct
func (p *Pagser) doParse(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) (err error) {
	switch val.Kind() {
	case reflect.Pointer:
		return p.doParsePointer(val, stackValues, selection)
	case reflect.Struct:
		return p.doParseStruct(val, stackValues, selection)
	}

	return nil
}

func (p *Pagser) doParsePointer(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) (err error) {
	err = p.doParse(reflect.Indirect(val), stackValues, selection)
	if err != nil {
		return err
		// return fmt.Errorf("tag=`%v` %#v parser error: %v", tagValue, subModel, err)
	}
	return nil
}

func (p *Pagser) doParseStruct(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) (err error) {
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		fieldType := val.Type().Field(i)
		fieldKind := fieldType.Type.Kind()

		// tagValue := fieldType.Tag.Get(parserTagName)
		tagValue, tagOk := fieldType.Tag.Lookup(p.Config.TagName)
		if !tagOk {
			if p.Config.Debug {
				fmt.Printf("[INFO] not found tag name=[%v] in field: %v, eg: `%v:\".navlink a->attr(href)\"`\n",
					p.Config.TagName, fieldType.Name, p.Config.TagName)
			}
			continue
		}
		if tagValue == ignoreSymbol {
			continue
		}

		cacheTag, ok := p.mapTags.Load(tagValue)
		var tag *tagTokenizer
		if !ok || cacheTag == nil {
			tag, err = p.newTag(tagValue)
			if err != nil {
				return err
			}
			p.mapTags.Store(tagValue, tag)
		} else {
			tag = cacheTag.(*tagTokenizer)
		}

		node := selection
		if tag.Selector != "" {
			node = selection.Find(tag.Selector)
		}

		var callOutValue interface{}
		var callErr error
		if tag.FuncName != "" {
			callOutValue, callErr = p.findAndExecFunc(val, stackValues, tag, node)
			if callErr != nil {
				return fmt.Errorf("tag=`%v` parse func error: %v", tagValue, callErr)
			}
			if subNode, ok := callOutValue.(*goquery.Selection); ok {
				// set sub node to current node
				node = subNode
			} else {
				svErr := p.setRefectValue(fieldType.Type.Kind(), fieldValue, callOutValue)
				if svErr != nil {
					return fmt.Errorf("tag=`%v` set value error: %v", tagValue, svErr)
				}
				// goto parse next field
				continue
			}
		}

		if stackValues == nil {
			stackValues = make([]reflect.Value, 0)
		}
		stackValues = append(stackValues, val)

		// set value
		switch {
		case fieldKind == reflect.Ptr:
			subModel := reflect.New(fieldType.Type.Elem())
			fieldValue.Set(subModel)
			err = p.doParse(subModel, stackValues, node)
			if err != nil {
				return fmt.Errorf("tag=`%v` %#v parser error: %v", tagValue, subModel, err)
			}
			// Slice
		case fieldKind == reflect.Slice:
			sliceType := fieldValue.Type()
			itemType := sliceType.Elem()
			itemKind := itemType.Kind()
			slice := reflect.MakeSlice(sliceType, node.Size(), node.Size())
			node.EachWithBreak(func(i int, subNode *goquery.Selection) bool {
				// outhtml, _ := goquery.OuterHtml(subNode)
				// log.Printf("%v => %v", i, outhtml)
				itemValue := reflect.New(itemType).Elem()
				switch {
				case itemKind == reflect.Struct:
					err = p.doParse(itemValue.Addr(), stackValues, subNode)
					if err != nil {
						err = fmt.Errorf("tag=`%v` %#v parser error: %v", tagValue, itemValue, err)
						return false
					}
				case itemKind == reflect.Ptr && itemValue.Type().Elem().Kind() == reflect.Struct:
					itemValue = reflect.New(itemType.Elem())
					err = p.doParse(itemValue, stackValues, subNode)
					if err != nil {
						err = fmt.Errorf("tag=`%v` %#v parser error: %v", tagValue, itemValue, err)
						return false
					}
				default:
					itemValue.SetString(strings.TrimSpace(subNode.Text()))
				}
				slice.Index(i).Set(itemValue)
				return true
			})
			if err != nil {
				return err
			}
			fieldValue.Set(slice)
		case fieldKind == reflect.Struct:
			subModel := reflect.New(fieldType.Type)
			err = p.doParse(subModel, stackValues, node)
			if err != nil {
				return fmt.Errorf("tag=`%v` %#v parser error: %v", tagValue, subModel, err)
			}
			fieldValue.Set(subModel.Elem())
			// UnsafePointer
			// Complex64
			// Complex128
			// Array
			// Chan
			// Func
		default:
			fieldValue.SetString(strings.TrimSpace(node.Text()))
		}
	}

	return nil
}

func (p *Pagser) findAndExecFunc(val reflect.Value, stackValues []reflect.Value, selTag *tagTokenizer, node *goquery.Selection) (interface{}, error) {
	// If function not set, return node as tring
	if selTag.FuncName == "" {
		return strings.TrimSpace(node.Text()), nil
	}

	// Try to find function in the methods of the value or its pointer, calling it if found
	callMethod := findMethod(val, selTag.FuncName)
	if callMethod.IsValid() {
		return execMethod(callMethod, selTag, node)
	}

	// Try to find function in the methods of the parent values or their pointers, calling it if found
	size := len(stackValues)
	if size > 0 {
		for i := size - 1; i >= 0; i-- {
			callMethod = findMethod(stackValues[i], selTag.FuncName)
			if callMethod.IsValid() {
				return execMethod(callMethod, selTag, node)
			}
		}
	}

	// Try to find function in the globally registered functions, calling it if found
	if fn, ok := p.mapFuncs.Load(selTag.FuncName); ok {
		cfn := fn.(CallFunc)
		outValue, err := cfn(node, selTag.FuncParams...)
		if err != nil {
			return nil, fmt.Errorf("call registered func %v error: %v", selTag.FuncName, err)
		}
		return outValue, nil
	}

	return nil, fmt.Errorf("method not found: %v", selTag.FuncName)
}

// findMethod finds a function in the methods of a value or its pointer.
// Value passed should not be a pointer.
// If function is not found a zero value will be returned
func findMethod(val reflect.Value, funcName string) reflect.Value {
	// Try to find method on value
	callMethod := val.MethodByName(funcName)
	if callMethod.IsValid() {
		return callMethod
	}

	// Try to find method on pointer to value
	if val.CanAddr() {
		valPtr := val.Addr()
		callMethod = valPtr.MethodByName(funcName)
		if callMethod.IsValid() {
			return callMethod
		}
	}

	// If method still not found, return a zero value
	return reflect.Value{}
}

func execMethod(callMethod reflect.Value, selTag *tagTokenizer, node *goquery.Selection) (interface{}, error) {
	callParams := make([]reflect.Value, 0)
	callParams = append(callParams, reflect.ValueOf(node))

	callReturns := callMethod.Call(callParams)
	if len(callReturns) <= 0 {
		return nil, fmt.Errorf("method %v not return any value", selTag.FuncName)
	}
	if len(callReturns) > 1 {
		if err, ok := callReturns[len(callReturns)-1].Interface().(error); ok {
			if err != nil {
				return nil, fmt.Errorf("method %v return error: %v", selTag.FuncName, err)
			}
		}
	}
	return callReturns[0].Interface(), nil
}

func (p Pagser) setRefectValue(kind reflect.Kind, fieldValue reflect.Value, v interface{}) (err error) {
	// set value
	switch {
	// Bool
	case kind == reflect.Bool:
		if p.Config.CastError {
			kv, err := cast.ToBoolE(v)
			if err != nil {
				return err
			}
			fieldValue.SetBool(kv)
		} else {
			fieldValue.SetBool(cast.ToBool(v))
		}
	case kind >= reflect.Int && kind <= reflect.Int64:
		if p.Config.CastError {
			kv, err := cast.ToInt64E(v)
			if err != nil {
				return err
			}
			fieldValue.SetInt(kv)
		} else {
			fieldValue.SetInt(cast.ToInt64(v))
		}
	case kind >= reflect.Uint && kind <= reflect.Uintptr:
		if p.Config.CastError {
			kv, err := cast.ToUint64E(v)
			if err != nil {
				return err
			}
			fieldValue.SetUint(kv)
		} else {
			fieldValue.SetUint(cast.ToUint64(v))
		}
	case kind == reflect.Float32 || kind == reflect.Float64:
		if p.Config.CastError {
			value, err := cast.ToFloat64E(v)
			if err != nil {
				return err
			}
			fieldValue.SetFloat(value)
		} else {
			fieldValue.SetFloat(cast.ToFloat64(v))
		}
	case kind == reflect.String:
		if p.Config.CastError {
			kv, err := cast.ToStringE(v)
			if err != nil {
				return err
			}
			fieldValue.SetString(kv)
		} else {
			fieldValue.SetString(cast.ToString(v))
		}
	case kind == reflect.Slice || kind == reflect.Array:
		sliceType := fieldValue.Type().Elem()
		itemKind := sliceType.Kind()
		if p.Config.CastError {
			switch itemKind {
			case reflect.Bool:
				kv, err := cast.ToBoolSliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Int:
				kv, err := cast.ToIntSliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Int32:
				kv, err := toInt32SliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Int64:
				kv, err := toInt64SliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Float32:
				kv, err := toFloat32SliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Float64:
				kv, err := toFloat64SliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.String:
				kv, err := cast.ToStringSliceE(v)
				if err != nil {
					return err
				}
				fieldValue.Set(reflect.ValueOf(kv))
			default:
				fieldValue.Set(reflect.ValueOf(v))
			}
		} else {
			switch itemKind {
			case reflect.Bool:
				kv := cast.ToBoolSlice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Int:
				kv := cast.ToIntSlice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Int32:
				kv := toInt32Slice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Int64:
				kv := toInt64Slice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Float32:
				kv := toFloat32Slice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.Float64:
				kv := toFloat64Slice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			case reflect.String:
				kv := cast.ToStringSlice(v)
				fieldValue.Set(reflect.ValueOf(kv))
			default:
				fieldValue.Set(reflect.ValueOf(v))
			}
		}
	// case kind == reflect.Interface:
	//	fieldValue.Set(reflect.ValueOf(v))
	default:
		fieldValue.Set(reflect.ValueOf(v))
		// return fmt.Errorf("not support type %v", kind)
	}
	return nil
}
