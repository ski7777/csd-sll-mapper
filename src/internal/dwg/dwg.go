package dwg

/*
#include <stdlib.h>
#include "dwg.h"
#include "dynapi.h"

static int get_attrib_utf8(
    void *attrib,
    const char *field,
    char **out,
    int *isnew
) {
    return dwg_dynapi_entity_utf8text(
        attrib,
        "ATTRIB",
        field,
        out,
        isnew,
        NULL
    );
}
*/
import "C"

import (
	"errors"
	"math"
	"slices"
	"unsafe"

	libredwg "github.com/ski7777/golibredwg"
)

type dwgError struct {
	Flag uint
	Name string
}

var dwgErrors = []dwgError{
	{1, "DWG_ERR_WRONGCRC"},
	{2, "DWG_ERR_NOTYETSUPPORTED"},
	{4, "DWG_ERR_UNHANDLEDCLASS"},
	{8, "DWG_ERR_INVALIDTYPE"},
	{16, "DWG_ERR_INVALIDHANDLE"},
	{32, "DWG_ERR_INVALIDEED"},
	{64, "DWG_ERR_VALUEOUTOFBOUNDS"},
	{128, "DWG_ERR_CLASSESNOTFOUND"},
	{256, "DWG_ERR_SECTIONNOTFOUND"},
	{512, "DWG_ERR_PAGENOTFOUND"},
	{1024, "DWG_ERR_INTERNALERROR"},
	{2048, "DWG_ERR_INVALIDDWG"},
	{4096, "DWG_ERR_IOERROR"},
	{8192, "DWG_ERR_OUTOFMEM"},
}

func getDWGErrors(rc int) (errs []error, warnings []error) {
	if rc == 0 {
		return
	}

	for _, err := range dwgErrors {
		if uint(rc)&err.Flag != 0 {
			if err.Flag >= 128 {
				errs = append(errs, errors.New(err.Name))
			} else {
				warnings = append(warnings, errors.New(err.Name))
			}
		}
	}
	return
}

func getAttribText(
	attrib libredwg.Dwg_Entity_ATTRIB,
	field string,
) (string, bool) {
	if attrib == nil {
		return "", false
	}

	cField := C.CString(field)
	defer C.free(unsafe.Pointer(cField))

	var cText *C.char
	var isNew C.int

	ok := C.get_attrib_utf8(
		unsafe.Pointer(attrib.Swigcptr()),
		cField,
		&cText,
		&isNew,
	)

	if ok == 0 {
		return "", false
	}

	if cText == nil {
		return "", true
	}

	text := C.GoString(cText)

	/*
	 * LibreDWG sets isnew when the UTF-8 string is a newly
	 * allocated copy. Free it after copying it into Go.
	 */
	if isNew != 0 {
		C.free(unsafe.Pointer(cText))
	}

	return text, true
}

func getAttributes(
	dwg libredwg.Dwg_Data,
	insert libredwg.Dwg_Entity_INSERT,
) (attrs map[string]string, err error) {
	attrs = make(map[string]string)
	attribs := insert.GetAttribs()

	if attribs == nil {
		return
	}

	base := attribs.Swigcptr()

	if base == 0 {
		return
	}

	numOwned := insert.GetNum_owned()

	if numOwned == 0 {
		return
	}

	refs := unsafe.Slice(
		(*uintptr)(unsafe.Pointer(base)),
		int(numOwned),
	)

	attributeIndex := 0

	for _, p := range refs {
		if p == 0 {
			continue
		}

		ref := libredwg.Dwg_Object_Ref(
			libredwg.SwigcptrDwg_Object_Ref(p),
		)

		handle := ref.GetAbsolute_ref()

		if handle == 0 {
			continue
		}

		obj := libredwg.Dwg_resolve_handle(
			dwg,
			handle,
		)

		if obj == nil {
			continue
		}

		attrib := libredwg.Dwg_object_to_ATTRIB(obj)

		if attrib == nil {
			continue
		}

		tag, tagOK := getAttribText(
			attrib,
			"tag",
		)

		value, valueOK := getAttribText(
			attrib,
			"text_value",
		)

		if !tagOK && !valueOK {
			err = errors.New("tag or text_value lookup failed")
			return
		}

		attrs[tag] = value

		attributeIndex++
	}
	return
}

type Object struct {
	BlockName  string
	X, Y       float64
	Rotation   float64
	Attributes map[string]string
}

func LoadDWG(filename string, blocknames []string) (objs []Object, err error, warnings []error) {
	dwg := libredwg.NewDwg_Data()

	if dwg == nil {
		err = errors.New("failed to allocate Dwg_Data")
		return
	}

	rc := libredwg.Dwg_read_file(
		filename,
		dwg,
	)

	errs, warns := getDWGErrors(rc)
	warnings = append(warnings, warns...)
	if len(errs) > 0 {
		err = errors.Join(append([]error{errors.New("Failed reading dwg")}, errs...)...)
		return
	}
	// ToDo: print warnings if any

	objects := dwg.GetObject()
	numObjects := dwg.GetNum_objects()

	for i := int64(0); i < int64(numObjects); i++ {
		obj := libredwg.Dwg_Object_Array_getitem(
			objects,
			i,
		)

		if obj == nil {
			continue
		}

		if libredwg.Dwg_object_get_type(obj) !=
			int(libredwg.DWG_TYPE_INSERT) {
			continue
		}

		insert := libredwg.Dwg_object_to_INSERT(obj)

		if insert == nil {
			continue
		}

		blockRef := insert.GetBlock_header()

		if blockRef == nil {
			continue
		}

		var nameErr int

		name := libredwg.Dwg_ref_get_table_name(
			blockRef,
			&nameErr,
		)

		if !slices.Contains(blocknames, name) {
			continue
		}

		o := Object{
			BlockName: name,
		}

		pt := insert.GetIns_pt()
		o.X = pt.GetX()
		o.Y = pt.GetY()

		rotation := insert.GetRotation()
		o.Rotation = rotation * 180.0 / math.Pi

		if insert.GetHas_attribs() != 0 {
			o.Attributes, err = getAttributes(
				dwg,
				insert,
			)
			if err != nil {
				return
			}
		}

		objs = append(objs, o)
	}

	libredwg.Dwg_free(dwg)

	return
}
