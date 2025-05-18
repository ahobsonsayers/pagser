package pagser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

const rawParseHtml = `
<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <title>Pagser Example</title>
	<meta name="keywords" content="golang,pagser,goquery,html,page,parser,colly">
</head>

<body>
	<h1><u>Pagser</u> H1 Title</h1>
	<div class="navlink">
		<div class="container">
			<ul class="clearfix">
				<li id=""><a href="/">Index</a></li>
				<li id="2"><a href="/list/web" title="web site">Web page</a></li>
				<li id="3"><a href="/list/pc" title="pc page">Pc Page</a></li>
				<li id="4"><a href="/list/mobile" title="mobile page">Mobile Page</a></li>
			</ul>
		</div>
	</div>
	<div class="words" show="true">A|B|C|D</div>
	
	<div class="group" id="a">
		<h2>Email</h2>
		<ul>
			<li class="item" id="1" name="email" value="pagser@foolin.github">pagser@foolin.github</li>
			<li class="item" id="2" name="email" value="pagser@foolin.github">hello@pagser.foolin</li>
		</ul>
	</div>
	<div class="group" id="b">
		<h2>Bool</h2>
		<ul>
			<li class="item" id="3" name="bool" value="true">true</li>
			<li class="item" id="4" name="bool" value="false">false</li>
		</ul>
	</div>
	<div class="group" id="c">
		<h2>Number</h2>
		<ul>
			<li class="item" id="5" name="number" value="12345">12345</li>
			<li class="item" id="6" name="number" value="67890">67890</li>
		</ul>
	</div>
	<div class="group" id="d">
		<h2>Float</h2>
		<ul>
			<li class="item" id="7" name="float" value="123.45">123.45</li>
			<li class="item" id="8" name="float" value="678.90">678.90</li>
		</ul>
	</div>
	<div class="empty">
		<span></span><span></span>
	<div>
</body>
</html>
`

const expectedParseDataJson = `
{
  "Title": "Pagser Example",
  "Keywords": [
    "golang",
    "pagser",
    "goquery",
    "html",
    "page",
    "parser",
    "colly"
  ],
  "H1": "Pagser H1 Title",
  "H1Text": "Pagser H1 Title",
  "H1TextEmpty": "Pagser H1 Title",
  "TextEmptyNoData": "nodata",
  "H1Html": "<u>Pagser</u> H1 Title",
  "H1OutHtml": "<h1><u>Pagser</u> H1 Title</h1>",
  "SameFuncValue": "Struct-Same-Func-Pagser H1 Title",
  "MyGlobalFuncValue": "Global-Pagser H1 Title",
  "MyStructFuncValue": "Struct-Pagser H1 Title",
  "FillFieldFuncValue": "FillFieldFunc-Pagser H1 Title",
  "FillFieldOtherValue": "This value is set by the FillFieldFunc() function -Pagser H1 Title",
  "NavList": [
    {
      "ID": -1,
      "Link": {"Name": "Index", "Url": "/", "AbsUrl": "https://thisvar.com/"},
      "LinkHtml": "Index",
      "ParentFuncName": "ParentFunc-Index"
    },
    {
      "ID": 2,
      "Link": {
        "Name": "Web page",
        "Url": "/list/web",
        "AbsUrl": "https://thisvar.com/list/web"
      },
      "LinkHtml": "Web page",
      "ParentFuncName": "ParentFunc-Web page"
    },
    {
      "ID": 3,
      "Link": {
        "Name": "Pc Page",
        "Url": "/list/pc",
        "AbsUrl": "https://thisvar.com/list/pc"
      },
      "LinkHtml": "Pc Page",
      "ParentFuncName": "ParentFunc-Pc Page"
    },
    {
      "ID": 4,
      "Link": {
        "Name": "Mobile Page",
        "Url": "/list/mobile",
        "AbsUrl": "https://thisvar.com/list/mobile"
      },
      "LinkHtml": "Mobile Page",
      "ParentFuncName": "ParentFunc-Mobile Page"
    }
  ],
  "NavFirst": {"ID": 0, "Name": "Index", "Url": "/"},
  "NavLast": {"ID": 4, "Name": "Mobile Page", "Url": "/list/mobile"},
  "SubStruct": {
    "Label": "",
    "Values": ["pagser@foolin.github", "pagser@foolin.github"]
  },
  "SubPtrStruct": {"Label": "", "Values": []},
  "NavFirstID": 0,
  "NavLastID": 4,
  "NavLastData": "nodata",
  "NavFirstIDDefaultValue": -999,
  "NavTextList": ["Index", "Web page", "Pc Page", "Mobile Page"],
  "NavEachText": ["Index", "Web page", "Pc Page", "Mobile Page"],
  "NavEachTextEmpty": ["Index", "Web page", "Pc Page", "Mobile Page"],
  "NavEachTextEmptyNoData": ["nodata", "nodata"],
  "NavEachAttrID": ["", "2", "3", "4"],
  "NavEachAttrEmptyID": ["-1", "2", "3", "4"],
  "NavEachHtml": [
    "<a href=\"/\">Index</a>",
    "<a href=\"/\">Index</a>",
    "<a href=\"/\">Index</a>",
    "<a href=\"/\">Index</a>"
  ],
  "NavEachOutHtml": [
    "<li id=\"\"><a href=\"/\">Index</a></li>",
    "<li id=\"\"><a href=\"/\">Index</a></li>",
    "<li id=\"\"><a href=\"/\">Index</a></li>",
    "<li id=\"\"><a href=\"/\">Index</a></li>"
  ],
  "NavJoinString": "Index|Web page|Pc Page|Mobile Page",
  "NavEqText": "Web page",
  "NavEqAttr": "2",
  "NavEqHtml": "<a href=\"/list/web\" title=\"web site\">Web page</a>",
  "NavEqOutHtml": "<li id=\"2\"><a href=\"/list/web\" title=\"web site\">Web page</a></li>",
  "NavSize": 4,
  "SubPageData": {
    "Text": "Mobile Page",
    "SubFuncValue": "SubFunc-Mobile Page",
    "ParentFuncValue": "ParentFunc-Mobile Page",
    "SameFuncValue": "Sub-Struct-Same-Func-Mobile Page"
  },
  "SubPageDataList": [
    {
      "Text": "Index",
      "SubFuncValue": "SubFunc-Index",
      "ParentFuncValue": "ParentFunc-Index",
      "SameFuncValue": "Sub-Struct-Same-Func-Index"
    },
    {
      "Text": "Web page",
      "SubFuncValue": "SubFunc-Web page",
      "ParentFuncValue": "ParentFunc-Web page",
      "SameFuncValue": "Sub-Struct-Same-Func-Web page"
    },
    {
      "Text": "Pc Page",
      "SubFuncValue": "SubFunc-Pc Page",
      "ParentFuncValue": "ParentFunc-Pc Page",
      "SameFuncValue": "Sub-Struct-Same-Func-Pc Page"
    },
    {
      "Text": "Mobile Page",
      "SubFuncValue": "SubFunc-Mobile Page",
      "ParentFuncValue": "ParentFunc-Mobile Page",
      "SameFuncValue": "Sub-Struct-Same-Func-Mobile Page"
    }
  ],
  "WordsSplitArray": ["A", "B", "C", "D"],
  "WordsSplitArrayNoTrim": ["A", "B", "C", "D"],
  "WordsShow": true,
  "WordsConcatText": "this is words:[A|B|C|D]",
  "WordsConcatAttr": "isShow = [true]",
  "Email": "pagser@foolin.github",
  "Emails": ["pagser@foolin.github", "pagser@foolin.github"],
  "CastBoolValue": true,
  "CastBoolNoExist": false,
  "CastBoolArray": [true, false],
  "CastIntValue": 12345,
  "CastIntNoExist": -1,
  "CastIntArray": [12345, 67890],
  "CastInt32Value": 12345,
  "CastInt32NoExist": -1,
  "CastInt32Array": [12345, 67890],
  "CastInt64Value": 12345,
  "CastInt64NoExist": -1,
  "CastInt64Array": [12345, 67890],
  "CastFloat32Value": 123.45,
  "CastFloat32NoExist": 0,
  "CastFloat32Array": [123.45, 678.9],
  "CastFloat64Value": 123.45,
  "CastFloat64NoExist": 0,
  "CastFloat64Array": [123.45, 678.9],
  "NodeChild": [
    {"Value": "Email"},
    {"Value": "pagser@foolin.github\n\t\t\thello@pagser.foolin"},
    {"Value": "Bool"},
    {"Value": "true\n\t\t\tfalse"},
    {"Value": "Number"},
    {"Value": "12345\n\t\t\t67890"},
    {"Value": "Float"},
    {"Value": "123.45\n\t\t\t678.90"}
  ],
  "NodeChildSelector": [
    {"Value": "Email"},
    {"Value": "Bool"},
    {"Value": "Number"},
    {"Value": "Float"}
  ],
  "NodeEqFirst": {"Value": "Email"},
  "NodeEqLast": {"Value": "Float"},
  "NodeEqPrev": [
    {"Value": "pagser@foolin.github"},
    {"Value": "true"},
    {"Value": "12345"},
    {"Value": "123.45"}
  ],
  "NodeEqPrevSelector": {"Value": "pagser@foolin.github"},
  "NodeEqNext": [
    {"Value": "hello@pagser.foolin"},
    {"Value": "false"},
    {"Value": "67890"},
    {"Value": "678.90"}
  ],
  "NodeEqNextSelector": {"Value": "hello@pagser.foolin"},
  "NodeParent": [
    {"Value": "Email"},
    {"Value": "Bool"},
    {"Value": "Number"},
    {"Value": "Float"}
  ],
  "NodeParents": [
    {"Value": "Email"},
    {"Value": "EmailBoolNumberFloat"},
    {"Value": "EmailBoolNumberFloat"},
    {"Value": "Bool"},
    {"Value": "Number"},
    {"Value": "Float"}
  ],
  "NodeParentsSelector": [{"Value": "Bool"}],
  "NodeParentsUntil": [
    {"Value": "Email"},
    {"Value": "EmailBoolNumberFloat"},
    {"Value": "EmailBoolNumberFloat"},
    {"Value": "Number"},
    {"Value": "Float"}
  ],
  "NodeParentSelector": [{"Value": "Email"}],
  "NodeEqSiblings": [
    {"Value": "hello@pagser.foolin"},
    {"Value": "false"},
    {"Value": "67890"},
    {"Value": "678.90"}
  ],
  "NodeEqSiblingsSelector": [{"Value": "hello@pagser.foolin"}]
}
`

type ParseData struct {
	Title               string   `pagser:"title"`
	Keywords            []string `pagser:"meta[name='keywords']->attrSplit(content)"`
	H1                  string   `pagser:"h1"`
	H1Text              string   `pagser:"h1->text()"`
	H1TextEmpty         string   `pagser:"h1->textEmpty('')"`
	TextEmptyNoData     string   `pagser:".empty->textEmpty('nodata')"`
	H1Html              string   `pagser:"h1->html()"`
	H1OutHtml           string   `pagser:"h1->outerHtml()"`
	SameFuncValue       string   `pagser:"h1->SameFunc()"`
	MyGlobalFuncValue   string   `pagser:"h1->MyGlobFunc()"`
	MyStructFuncValue   string   `pagser:"h1->MyStructFunc()"`
	FillFieldFuncValue  string   `pagser:"h1->FillFieldFunc()"`
	FillFieldOtherValue string   // Set value by FillFieldFunc()
	NavList             []struct {
		ID   int `pagser:"->attrEmpty(id, -1)"`
		Link struct {
			Name   string `pagser:"->text()"`
			Url    string `pagser:"->attr(href)"`
			AbsUrl string `pagser:"->absHref('https://thisvar.com')"`
		} `pagser:"a"`
		LinkHtml       string `pagser:"a->html()"`
		ParentFuncName string `pagser:"a->ParentFunc()"`
	} `pagser:".navlink li"`
	NavFirst struct {
		ID   int    `pagser:"->attrEmpty(id, 0)"`
		Name string `pagser:"a->text()"`
		Url  string `pagser:"a->attr(href)"`
	} `pagser:".navlink li->first()"`
	NavLast struct {
		ID   int    `pagser:"->attrEmpty(id, 0)"`
		Name string `pagser:"a->text()"`
		Url  string `pagser:"a->attr(href)"`
	} `pagser:".navlink li->last()"`
	SubStruct struct {
		Label  string   `pagser:"label"`
		Values []string `pagser:".item->eachAttr(value)"`
	} `pagser:".group->eq(0)"`
	SubPtrStruct *struct {
		Label  string   `pagser:"label"`
		Values []string `pagser:".item->eachAttr(value)"`
	} `pagser:".group:last-child"`
	NavFirstID             int            `pagser:".navlink li:first-child->attrEmpty(id, 0)"`
	NavLastID              uint           `pagser:".navlink li:last-child->attr(id)"`
	NavLastData            string         `pagser:".navlink li:last-child->attr(data, 'nodata')"`
	NavFirstIDDefaultValue int            `pagser:".navlink li:first-child->attrEmpty(id, -999)"`
	NavTextList            []string       `pagser:".navlink li"`
	NavEachText            []string       `pagser:".navlink li->eachText()"`
	NavEachTextEmpty       []string       `pagser:".navlink li->eachTextEmpty('')"`
	NavEachTextEmptyNoData []string       `pagser:".empty span->eachTextEmpty('nodata')"`
	NavEachAttrID          []string       `pagser:".navlink li->eachAttr(id)"`
	NavEachAttrEmptyID     []string       `pagser:".navlink li->eachAttrEmpty(id, -1)"`
	NavEachHtml            []string       `pagser:".navlink li->eachHtml()"`
	NavEachOutHtml         []string       `pagser:".navlink li->eachOutHtml()"`
	NavJoinString          string         `pagser:".navlink li->eachTextJoin(|)"`
	NavEqText              string         `pagser:".navlink li->eqAndText(1)"`
	NavEqAttr              string         `pagser:".navlink li->eqAndAttr(1, id)"`
	NavEqHtml              string         `pagser:".navlink li->eqAndHtml(1)"`
	NavEqOutHtml           string         `pagser:".navlink li->eqAndOutHtml(1)"`
	NavSize                int            `pagser:".navlink li->size()"`
	SubPageData            *SubPageData   `pagser:".navlink li:last-child"`
	SubPageDataList        []*SubPageData `pagser:".navlink li"`
	WordsSplitArray        []string       `pagser:".words->textSplit(|)"`
	WordsSplitArrayNoTrim  []string       `pagser:".words->textSplit('|', false)"`
	WordsShow              bool           `pagser:".words->attrEmpty(show, false)"`
	WordsConcatText        string         `pagser:".words->textConcat('this is words:', [, $value, ])"`
	WordsConcatAttr        string         `pagser:".words->attrConcat(show, 'isShow = [', $value, ])"`
	Email                  string         `pagser:".item[name='email']->attr('value')"`
	Emails                 []string       `pagser:".item[name='email']->eachAttrEmpty(value, '')"`
	CastBoolValue          bool           `pagser:".item[name='bool']->attrEmpty(value, false)"`
	CastBoolNoExist        bool           `pagser:".item[name='bool']->attrEmpty(value2, false)"`
	CastBoolArray          []bool         `pagser:".item[name='bool']->eachAttrEmpty(value, false)"`
	CastIntValue           int            `pagser:".item[name='number']->attrEmpty(value, 0)"`
	CastIntNoExist         int            `pagser:".item[name='number']->attrEmpty(value2, -1)"`
	CastIntArray           []int          `pagser:".item[name='number']->eachAttrEmpty(value, 0)"`
	CastInt32Value         int32          `pagser:".item[name='number']->attrEmpty(value, 0)"`
	CastInt32NoExist       int32          `pagser:".item[name='number']->attrEmpty(value2, -1)"`
	CastInt32Array         []int32        `pagser:".item[name='number']->eachAttrEmpty(value, 0)"`
	CastInt64Value         int64          `pagser:".item[name='number']->attrEmpty(value, 0)"`
	CastInt64NoExist       int64          `pagser:".item[name='number']->attrEmpty(value2, -1)"`
	CastInt64Array         []int64        `pagser:".item[name='number']->eachAttrEmpty(value, 0)"`
	CastFloat32Value       float32        `pagser:".item[name='float']->attrEmpty(value, 0)"`
	CastFloat32NoExist     float32        `pagser:".item[name='float']->attrEmpty(value2, 0.0)"`
	CastFloat32Array       []float32      `pagser:".item[name='float']->eachAttrEmpty(value, 0)"`
	CastFloat64Value       float64        `pagser:".item[name='float']->attrEmpty(value, 0)"`
	CastFloat64NoExist     float64        `pagser:".item[name='float']->attrEmpty(value2, 0.0)"`
	CastFloat64Array       []float64      `pagser:".item[name='float']->eachAttrEmpty(value, 0)"`
	NodeChild              []struct {
		Value string `pagser:"->text()"`
	} `pagser:".group->child()"`
	NodeChildSelector []struct {
		Value string `pagser:"->text()"`
	} `pagser:".group->child('h2')"`
	NodeEqFirst struct {
		Value string `pagser:"h2->text()"`
	} `pagser:".group->eq(0)"`
	NodeEqLast struct {
		Value string `pagser:"h2->text()"`
	} `pagser:".group->eq(-1)"`
	NodeEqPrev []struct {
		Value string `pagser:"->text()"`
	} `pagser:".item:last-child->prev()"`
	NodeEqPrevSelector struct {
		Value string `pagser:"->text()"`
	} `pagser:".item:last-child->prev('[id=\"1\"]')"`
	NodeEqNext []struct {
		Value string `pagser:"->text()"`
	} `pagser:".item:first-child->next()"`
	NodeEqNextSelector struct {
		Value string `pagser:"->text()"`
	} `pagser:".item:first-child->next('[id=\"2\"]')"`
	NodeParent []struct {
		Value string `pagser:"h2->text()"`
	} `pagser:"h2:first-child->parent()"`
	NodeParents []struct {
		Value string `pagser:"h2->text()"`
	} `pagser:"h2:first-child->parents()"`
	NodeParentsSelector []struct {
		Value string `pagser:"h2->text()"`
	} `pagser:"h2:first-child->parents('[id=\"b\"]')"`
	NodeParentsUntil []struct {
		Value string `pagser:"h2->text()"`
	} `pagser:"h2:first-child->parentsUntil('[id=\"b\"]')"`
	NodeParentSelector []struct {
		Value string `pagser:"h2->text()"`
	} `pagser:"h2:first-child->parent('[id=\"a\"]')"`
	NodeEqSiblings []struct {
		Value string `pagser:"->text()"`
	} `pagser:".item:first-child->siblings()"`
	NodeEqSiblingsSelector []struct {
		Value string `pagser:"->text()"`
	} `pagser:".item:first-child->siblings('[id=\"2\"]')"`
}

// this method will auto call, not need register.
func MyGlobalFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "Global-" + selection.Text(), nil
}

// this method will auto call, not need register.
func SameFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "Global-Same-Func-" + selection.Text(), nil
}

// this method will auto call, not need register.
func (pd ParseData) MyStructFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "Struct-" + selection.Text(), nil
}

func (pd ParseData) ParentFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "ParentFunc-" + selection.Text(), nil
}

// this method will auto call, not need register.
func (pd *ParseData) FillFieldFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	text := selection.Text()
	pd.FillFieldOtherValue = "This value is set by the FillFieldFunc() function -" + text
	return "FillFieldFunc-" + text, nil
}

func (pd *ParseData) SameFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "Struct-Same-Func-" + selection.Text(), nil
}

type SubPageData struct {
	Text            string `pagser:"->text()"`
	SubFuncValue    string `pagser:"->SubFunc()"`
	ParentFuncValue string `pagser:"->ParentFunc()"`
	SameFuncValue   string `pagser:"->SameFunc()"`
}

func (spd SubPageData) SubFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "SubFunc-" + selection.Text(), nil
}

func (spd SubPageData) SameFunc(selection *goquery.Selection, args ...string) (out interface{}, err error) {
	return "Sub-Struct-Same-Func-" + selection.Text(), nil
}

// Page parse from https://httpbin.org
type HttpBinData struct {
	Title       string `pagser:"title"`
	Version     string `pagser:".version->text()"`
	Description string `pagser:".description->text()"`
}

func TestParse(t *testing.T) {
	p := New()
	// register global function
	p.RegisterFunc("MyGlobFunc", MyGlobalFunc)
	p.RegisterFunc("SameFunc", SameFunc)

	var data ParseData
	err := p.Parse(&data, rawParseHtml)
	require.NoError(t, err)

	parseDataJson, err := json.Marshal(data)
	require.NoError(t, err)

	require.JSONEq(t, expectedParseDataJson, string(parseDataJson))

	fmt.Printf("json: %v\n", prettyJson(data))
}

func TestParse_TargetNotStruct(t *testing.T) {
	p := New()
	// register global function
	p.RegisterFunc("MyGlobFunc", MyGlobalFunc)
	p.RegisterFunc("SameFunc", SameFunc)

	var target string
	err := p.Parse(&target, rawParseHtml)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a struct")
}

func TestPagser_ParseDocument(t *testing.T) {
	cfg := Config{
		TagName:    "pagser",
		FuncSymbol: "->",
		CastError:  true,
		Debug:      true,
	}
	p, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// register global function
	p.RegisterFunc("MyGlobFunc", MyGlobalFunc)
	p.RegisterFunc("SameFunc", SameFunc)

	var data ParseData
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawParseHtml))
	if err != nil {
		t.Fatal(err)
	}
	err = p.ParseDocument(&data, doc)
	// err = p.ParseSelection(&data, doc.Selection)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("json: %v\n", prettyJson(data))
}

func TestPagser_ParseReader(t *testing.T) {
	res, err := http.Get("https://httpbin.org")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	p := New()

	var data HttpBinData
	err = p.ParseReader(&data, res.Body)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("json: %v\n", prettyJson(data))
}

func TestPagser_RegisterFunc(t *testing.T) {
	threads := 1000
	p := New()
	var wg sync.WaitGroup

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				for k, v := range builtinFuncs {
					p.RegisterFunc(k, v)
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
}
