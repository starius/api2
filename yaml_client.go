package api2

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"text/template"

	"github.com/starius/api2/typegen"
)

const yamlRoutes = `{{- range $key, $services := .}}
{{$key}}:
 {{- range $service, $methods := $services }}
  {{$service}}:
   {{- range $info := $methods}}
    {{$info.FnInfo.Method}}:
      method: "{{.Method}}"
      path: "{{.Path}}"{{end}}
	{{- end}}
{{- end}}
`

var yamlClientTemplate = template.Must(template.New("ts_static_client").Parse(yamlRoutes))

type YamlTypesGenConfig struct {
	OutDir         string
	ClientTemplate *template.Template
	Routes         []interface{}
	Blacklist      []BlacklistItem
}

func GenerateYamlClient(options *YamlTypesGenConfig) {
	if options.ClientTemplate == nil {
		options.ClientTemplate = yamlClientTemplate
	}
	_ = os.RemoveAll(filepath.Join(options.OutDir, "routes.yaml"))
	err := os.MkdirAll(options.OutDir, os.ModePerm)
	panicIf(err)
	typesFile, err := os.OpenFile(filepath.Join(options.OutDir, "routes.yaml"), os.O_WRONLY|os.O_CREATE, 0755)
	panicIf(err)
	parser := typegen.NewParser()
	parser.CustomParse = CustomParse
	allRoutes := []Route{}
	for _, getRoutes := range options.Routes {
		genValue := reflect.ValueOf(getRoutes)
		serviceArg := reflect.New(genValue.Type().In(0)).Elem()
		routesValues := genValue.Call([]reflect.Value{serviceArg})
		routes := routesValues[0].Interface().([]Route)
		allRoutes = append(allRoutes, routes...)
	}
	genYamlRoutes(typesFile, allRoutes, parser, options)

}

func genYamlRoutes(w io.Writer, routes []Route, p *typegen.Parser, options *YamlTypesGenConfig) {
	type routeDef struct {
		Method      string
		Path        string
		ReqType     string
		ResType     string
		Handler     interface{}
		FnInfo      FnInfo
		TypeInfoReq string
		TypeInfoRes string
	}
	m := map[string]map[string][]routeDef{}
OUTER:
	for _, route := range routes {
		handler := route.Handler
		if f, ok := handler.(funcer); ok {
			handler = f.Func()
		}

		handlerVal := reflect.ValueOf(handler)
		handlerType := handlerVal.Type()
		req := reflect.TypeOf(reflect.New(handlerType.In(1)).Elem().Interface()).Elem()
		response := reflect.TypeOf(reflect.New(handlerType.Out(0)).Elem().Interface()).Elem()
		fnInfo := GetFnInfo(route.Handler)
		for _, v := range options.Blacklist {
			if Matches(&v, fnInfo.PkgName, fnInfo.StructName, fnInfo.Method) {
				continue OUTER
			}
		}
		p.Parse(req, response)
		TypeInfoReq, err := serializeTypeInfo(prepare(req))
		panicIf(err)
		TypeInfoRes, err := serializeTypeInfo(prepare(response))
		panicIf(err)
		r := routeDef{
			ReqType:     req.String(),
			ResType:     response.String(),
			Method:      route.Method,
			Path:        route.Path,
			Handler:     route.Handler,
			FnInfo:      fnInfo,
			TypeInfoReq: string(TypeInfoReq),
			TypeInfoRes: string(TypeInfoRes),
		}

		if _, ok := m[fnInfo.PkgName]; !ok {
			m[fnInfo.PkgName] = make(map[string][]routeDef)
		}
		m[fnInfo.PkgName][fnInfo.StructName] = append(m[fnInfo.PkgName][fnInfo.StructName], r)
	}

	err := yamlClientTemplate.Execute(w, m)
	if err != nil {
		panic(err)
	}
}
