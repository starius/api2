package types

import "fmt"

// WeekDay can be converted to string.
type WeekDay int

const (
	SUNDAY WeekDay = 1
	MONDAY WeekDay = 2
)

// helllo this is week day
type WeekDay2 string

const (
	A  WeekDay2 = "3"
	A2 WeekDay2 = "4"
)

// WeekDay3 has no String method.
type WeekDay3 int

const (
	TUESDAY   WeekDay3 = 5
	WEDNESDAY WeekDay3 = 6
)

func (e WeekDay) String() string {
	switch e {
	case SUNDAY:
		return "sun"
	case MONDAY:
		return "mon"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

// Super custom array
type CustomArray []int

// struct level comment
type T struct {
	//body comment

	W WeekDay `json:"weekday"` // field level comment
	// kek
	W2 WeekDay2 `json:"weekday2"` // kek
	W3 WeekDay3 `json:"weekday3"`
}

type UserTag struct {
	Tag string `json:"tag"`
}

// Hello
type M struct {
	// body comment
	Username string `json:"Username2"` // field doc
}

// InternalStatus is a type used only inside ts-ignored fields.
type InternalStatus string

const (
	InternalStatusActive   InternalStatus = "active"
	InternalStatusInactive InternalStatus = "inactive"
)

// InternalDetail is a struct used only inside ts-ignored fields.
type InternalDetail struct {
	Status InternalStatus `json:"status"`
}

// MultilineEnum has a multi-line godoc comment.
// The second line must not land uncommented in gen.ts.
type MultilineEnum string

const (
	MultilineEnumA MultilineEnum = "a"
	MultilineEnumB MultilineEnum = "b"
)

// MultilineArray has a multi-line godoc comment.
// The second line must not land uncommented in gen.ts.
type MultilineArray []int

// WithTsIgnoredField has a field excluded from TypeScript but present on the wire.
type WithTsIgnoredField struct {
	Name     string `json:"name"`
	Internal string `json:"internal" ts:"-"`
}

// WithTsIgnoredComplexField has a ts-ignored field whose type has subtypes.
// Those subtypes should NOT appear in TypeScript output.
type WithTsIgnoredComplexField struct {
	Name   string         `json:"name"`
	Detail *InternalDetail `json:"detail,omitempty" ts:"-"`
}

// user
type User struct {
	M
	NestedStruct struct {
		UserTag
	}
	CustomArray *CustomArray `json:"k"`
	FirstName   *string      `json:"firstname"`
	SecondName  string       `json:"secondName"` // hello
	Tags        []UserTag    `json:"tags"`       // dima
}
