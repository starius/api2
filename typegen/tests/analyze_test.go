package typegen

import (
	"os"
	"reflect"
	"testing"

	gots "github.com/starius/api2/typegen"
	"github.com/starius/api2/typegen/tests/types"
)

func TestV2(t *testing.T) {
	p := gots.NewFromTypes(&types.User{}, &types.T{})
	gots.PrintTsTypes(p, os.Stdout, func(t reflect.Type) string {
		return ""
	})
}

// func TestRenderEnums(t *testing.T) {
// 	s := go2typings.New()
// 	s.Add(types.T{})

// 	out := strings.Builder{}
// 	err := s.RenderTo(&out)
// 	if err != nil {
// 		panic(err)
// 	}

// 	outString := out.String()
// 	expected := `export namespace types {
//   //github.com/zmitry/go2typings/tests/types.WeekDay
//   export type WeekDay = "mon" | "sun"
//   //github.com/zmitry/go2typings/tests/types.WeekDay2
//   export type WeekDay2 = "3" | "4"
//   //github.com/zmitry/go2typings/tests/types.WeekDay3
//   export type WeekDay3 = 5 | 6
//   //github.com/zmitry/go2typings/tests/types.T
//   export interface T {
//     weekday: types.WeekDay;
//     weekday2: types.WeekDay2;
//     weekday3: types.WeekDay3;
//   }
// }

// `

// 	if expected != outString {
// 		t.Fatalf("wrong output\ngot:\n'%s'\nwant:\n'%s'", outString, expected)
// 	}
// }
