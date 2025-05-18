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
func (p *Pagser) Parse(v interface{}, document string) error {
	reader, err := goquery.NewDocumentFromReader(strings.NewReader(document))
	if err != nil {
		return err
	}
	return p.ParseDocument(v, reader)
}

// ParseReader parse html to struct
func (p *Pagser) ParseReader(v interface{}, reader io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}
	return p.ParseDocument(v, doc)
}

// ParseDocument parse document to struct
func (p *Pagser) ParseDocument(v interface{}, document *goquery.Document) error {
	return p.ParseSelection(v, document.Selection)
}

// ParseSelection parse selection to struct
func (p *Pagser) ParseSelection(v interface{}, selection *goquery.Selection) error {
	val := reflect.ValueOf(v)

	// Check value is a pointer
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("%v is non-pointer", val.Type())
	}

	// Check pointer is not nil
	if val.IsNil() {
		return fmt.Errorf("%v is nil", val.Type())
	}

	// Check underlying type is a struct
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("%v is not a struct", elem.Type())
	}

	// Parse into pointer value
	return p.doParse(val, nil, selection)
}

// ParseSelection parse selection to struct
func (p *Pagser) doParse(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) error {
	switch val.Kind() {
	case reflect.Pointer:
		return p.doParsePointer(val, stackValues, selection)
	case reflect.Struct:
		return p.doParseStruct(val, stackValues, selection)
	case reflect.Slice:
		return p.doParseSlice(val, stackValues, selection)
	default:
		// UnsafePointer
		// Complex64
		// Complex128
		// Array
		// Chan
		// Func
		val.SetString(strings.TrimSpace(selection.Text()))
	}

	return nil
}

func (p *Pagser) doParsePointer(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) error {
	// If the pointer value is nil, create a new non-nil pointer to the underlying type
	if val.IsNil() {
		underlyingType := val.Type().Elem()
		newPtr := reflect.New(underlyingType)
		val.Set(newPtr)
	}

	// Parse into underlying value
	err := p.doParse(reflect.Indirect(val), stackValues, selection)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pagser) doParseStruct(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) error {
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		fieldType := val.Type().Field(i)

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
		var err error
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
				svErr := p.setFieldValue(fieldValue, callOutValue)
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

		// Do parse on struct field
		err = p.doParse(fieldValue, stackValues, node)
		if err != nil {
			return fmt.Errorf("tag=`%v` %#v parser error: %v", tagValue, fieldValue, err)
		}
	}
	return nil
}

func (p *Pagser) doParseSlice(val reflect.Value, stackValues []reflect.Value, selection *goquery.Selection) error {
	// Get slice to parse into, creating a new one if it is nil
	slice := val
	var newSlice bool
	if slice.IsNil() {
		slice = reflect.MakeSlice(val.Type(), selection.Size(), selection.Size())
		newSlice = true
	}

	// Parse into slice
	var err error
	selection.EachWithBreak(func(i int, subNode *goquery.Selection) bool {
		// Do parse on slice item
		itemValue := slice.Index(i)
		err = p.doParse(itemValue, stackValues, subNode)
		return err == nil
	})
	if err != nil {
		return err
	}

	// If we created a new slice, we need to set the val to it
	if newSlice {
		val.Set(slice)
	}

	return nil
}

func (p *Pagser) setFieldValue(fieldValue reflect.Value, value interface{}) error {
	var castValueInterface any
	var err error
	switch fieldValue.Kind() {
	case reflect.Bool:
		castValueInterface, err = cast.ToBoolE(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		castValueInterface, err = cast.ToInt64E(value)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		castValueInterface, err = cast.ToUint64E(value)

	case reflect.Float32, reflect.Float64:
		castValueInterface, err = cast.ToFloat64E(value)

	case reflect.String:
		castValueInterface, err = cast.ToStringE(value)

	case reflect.Slice, reflect.Array:
		// Run nested switch on item type
		switch fieldValue.Type().Elem().Kind() {
		case reflect.Bool:
			castValueInterface, err = cast.ToBoolSliceE(value)
		case reflect.Int:
			castValueInterface, err = cast.ToIntSliceE(value)
		case reflect.Int32:
			castValueInterface, err = toInt32SliceE(value)
		case reflect.Int64:
			castValueInterface, err = toInt64SliceE(value)
		case reflect.Float32:
			castValueInterface, err = toFloat32SliceE(value)
		case reflect.Float64:
			castValueInterface, err = toFloat64SliceE(value)
		case reflect.String:
			castValueInterface, err = cast.ToStringSliceE(value)
		default:
			castValueInterface = value
		}
	default:
		castValueInterface = value
	}
	if err != nil && p.Config.CastError {
		return err
	}

	// Get the reflect value of cast value, converting it if required
	castReflectValue := reflect.ValueOf(castValueInterface)
	fieldType := fieldValue.Type()
	if castReflectValue.Type() != fieldType && castReflectValue.CanConvert(fieldType) {
		castReflectValue = castReflectValue.Convert(fieldType)
	}

	fieldValue.Set(castReflectValue)

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
