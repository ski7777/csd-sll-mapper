package main

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
        "fmt"
        "log"
        "os"
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

func printDWGErrors(rc int) {
        if rc == 0 {
                fmt.Println("LibreDWG: no errors")
                return
        }

        fmt.Printf(
                "LibreDWG error mask: %d (0x%x)\n",
                rc,
                rc,
        )

        fmt.Println("Errors:")

        for _, err := range dwgErrors {
                if uint(rc)&err.Flag != 0 {
                        severity := "warning"

                        if err.Flag >= 128 {
                                severity = "ERROR"
                        }

                        fmt.Printf(
                                "  - [%s] %s (%d / 0x%x)\n",
                                severity,
                                err.Name,
                                err.Flag,
                                err.Flag,
                        )
                }
        }

        fmt.Println()
}

// getAttribText gets an ATTRIB string field as UTF-8.
//
// Do not use:
//   attrib.GetTag()
//   attrib.GetText_value()
//
// Those SWIG-generated getters directly cast BITCODE_TV to char*
// and use strlen(), which can truncate Unicode/TU strings.
//
// Instead we call LibreDWG's native UTF-8 conversion function.
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

func printAttributes(
        dwg libredwg.Dwg_Data,
        insert libredwg.Dwg_Entity_INSERT,
) {
        attribs := insert.GetAttribs()

        if attribs == nil {
                fmt.Println("    no attribs array")
                return
        }

        base := attribs.Swigcptr()

        if base == 0 {
                fmt.Println("    attribs array pointer is 0")
                return
        }

        numOwned := insert.GetNum_owned()

        if numOwned == 0 {
                fmt.Println("    num_owned is 0")
                return
        }

        /*
         * LibreDWG's INSERT structure contains:
         *
         *     BITCODE_H *attribs;
         *
         * The unmodified SWIG interface exposes this as a single
         * Dwg_Object_Ref instead of an array.
         *
         * Swigcptr() gives us the address of the underlying array.
         */
        refs := unsafe.Slice(
                (*uintptr)(unsafe.Pointer(base)),
                int(numOwned),
        )

        attributeIndex := 0

        for i, p := range refs {
                if p == 0 {
                        continue
                }

                ref := libredwg.Dwg_Object_Ref(
                        libredwg.SwigcptrDwg_Object_Ref(p),
                )

                handle := ref.GetAbsolute_ref()

                if handle == 0 {
                        fmt.Printf(
                                "    attrib[%d]: handle is 0\n",
                                i,
                        )
                        continue
                }

                /*
                 * Resolve the handle to the actual DWG object.
                 */
                obj := libredwg.Dwg_resolve_handle(
                        dwg,
                        handle,
                )

                if obj == nil {
                        fmt.Printf(
                                "    attrib[%d]: could not resolve handle 0x%x\n",
                                i,
                                handle,
                        )
                        continue
                }

                /*
                 * Convert the generic object into ATTRIB.
                 */
                attrib := libredwg.Dwg_object_to_ATTRIB(obj)

                if attrib == nil {
                        /*
                         * Not every entry necessarily has to be an ATTRIB.
                         */
                        continue
                }

                /*
                 * Retrieve the actual UTF-8 strings.
                 */
                tag, tagOK := getAttribText(
                        attrib,
                        "tag",
                )

                value, valueOK := getAttribText(
                        attrib,
                        "text_value",
                )

                fmt.Printf(
                        "    ATTRIB[%d] tag=%q text=%q",
                        attributeIndex,
                        tag,
                        value,
                )

                if !tagOK {
                        fmt.Print(" [tag lookup failed]")
                }

                if !valueOK {
                        fmt.Print(" [text_value lookup failed]")
                }

                fmt.Println()

                attributeIndex++
        }
}

func main() {
        if len(os.Args) != 2 {
                log.Fatalf(
                        "usage: %s drawing.dwg",
                        os.Args[0],
                )
        }

        filename := os.Args[1]

        fmt.Printf(
                "Reading DWG: %s\n",
                filename,
        )

        dwg := libredwg.NewDwg_Data()

        if dwg == nil {
                log.Fatal(
                        "failed to allocate Dwg_Data",
                )
        }

        rc := libredwg.Dwg_read_file(
                filename,
                dwg,
        )

        fmt.Printf(
                "dwg_read_file returned: %d (0x%x)\n",
                rc,
                rc,
        )

        printDWGErrors(rc)

        if rc >= 128 {
                libredwg.Dwg_free(dwg)

                log.Fatalf(
                        "DWG read failed",
                )
        }

        fmt.Println(
                "DWG loaded successfully (possibly with warnings)",
        )

        /*
         * Iterate over all objects.
         *
         * We deliberately don't use Dwg_getall_INSERT(), because
         * its Dwg_Entity_INSERT** return type is awkwardly exposed
         * by the unmodified SWIG interface.
         */
        objects := dwg.GetObject()
        numObjects := dwg.GetNum_objects()

        fmt.Printf(
                "DWG contains %d objects\n",
                numObjects,
        )

        for i := int64(0); i < int64(numObjects); i++ {
                obj := libredwg.Dwg_Object_Array_getitem(
                        objects,
                        i,
                )

                if obj == nil {
                        continue
                }

                /*
                 * DWG_TYPE_INSERT == 7.
                 */
                if libredwg.Dwg_object_get_type(obj) !=
                        int(libredwg.DWG_TYPE_INSERT) {
                        continue
                }

                insert := libredwg.Dwg_object_to_INSERT(obj)

                if insert == nil {
                        continue
                }

                /*
                 * Resolve the INSERT's block header.
                 */
                blockRef := insert.GetBlock_header()

                if blockRef == nil {
                        continue
                }

                var nameErr int

                /*
                 * Use LibreDWG's table-name resolver.
                 *
                 * This works correctly for your DWG, unlike the SWIG
                 * GetBlock_name()/GetName() string getters.
                 */
                name := libredwg.Dwg_ref_get_table_name(
                        blockRef,
                        &nameErr,
                )

/*              fmt.Printf(
                        "INSERT block=%q nameErr=%d\n",
                        name,
                        nameErr,
                )
*/
                if name != "Infostand" {
                        continue
                }

                /*
                 * INSERT insertion point.
                 */
                pt := insert.GetIns_pt()
rotation := insert.GetRotation()

fmt.Printf(
    "FOUND Stand: x=%f y=%f rotation=%f radians (%.2f°)\n",
    pt.GetX(),
    pt.GetY(),
    rotation,
    rotation*180.0/3.141592653589793,
)
                /*
                 * INSERT attributes.
                 */
                if insert.GetHas_attribs() != 0 {
                        printAttributes(
                                dwg,
                                insert,
                        )
                } else {
                        fmt.Println(
                                "    no attributes",
                        )
                }
        }

        libredwg.Dwg_free(dwg)

        fmt.Println("Done")
	}