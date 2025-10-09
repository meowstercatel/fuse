package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// StructInfo holds the struct name and original file name (without extension)
type StructInfo struct {
	Name     string // Struct name (e.g., "User")
	FileName string // File name without extension (e.g., "user")
	Msgid    string
}

// Template for the generated method (customizable)
var methodTmpl = `
package sessionmanager

import (
	"encoding/json"
	"fmt"
	"github.com/unknown321/fuse/message"
	"github.com/unknown321/fuse/tppmessage"
	"log/slog"
)

{{- if hasSuffix .Name "Response" }}
func Get{{.Name}}() tppmessage.{{.Name}} {
	t := tppmessage.{{.Name}}{}
	t.CryptoType = tppmessage.CRYPTO_TYPE_COMPOUND
	t.MsgID = tppmessage.{{.MsgID}}.String()
	t.Result = tppmessage.RESULT_NOERR
	t.Rqid = 0

	// code

	return t
}
{{- end }}

func Handle{{.Name}}(message *message.Message, override bool) error {
	if !override {
		return nil
	}

	slog.Info("using overridden version")
	var err error

	{{- if hasSuffix .Name "Response" }}
	t := Get{{.Name}}()
	{{- else }}
	t := tppmessage.{{.Name}}{}
	err = json.Unmarshal(message.MData, &t)
	if err != nil {
		return fmt.Errorf("cannot unmarshal: %w", err)
	}
	{{- end }}
	
	message.MData, err = json.Marshal(t)
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}

	return nil
}
`

func main() {
	dir := "../../tppmessage"

	// Collect struct names and file names
	var structs []StructInfo
	fset := token.NewFileSet()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		if strings.Contains(filepath.Dir(path), "hidden") {
			return nil
		}

		// Parse the .go file
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			log.Printf("Failed to parse %s: %v", path, err)
			return nil
		}

		// Extract file name without extension
		fileName := strings.TrimSuffix(info.Name(), ".go")

		// Find the first struct in the file
		for _, decl := range f.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if _, ok := typeSpec.Type.(*ast.StructType); ok {
							ss := strings.TrimSuffix(fileName, "_REQUEST")
							ss = strings.TrimSuffix(ss, "_RESPONSE")
							structs = append(structs, StructInfo{
								Name:     typeSpec.Name.Name,
								FileName: fileName,
								Msgid:    ss,
							})
							return nil // One struct per file
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	if len(structs) == 0 {
		log.Fatal("No structs found in the directory")
	}

	funcs := map[string]any{
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
	}

	// Generate a separate file for each struct
	tmpl := template.Must(template.New("method").Funcs(funcs).Parse(methodTmpl))
	for _, s := range structs {
		// Create output file named like the original (e.g., user_gen.go)
		outputFileName := "../" + s.FileName + ".go"
		outputFile, err := os.Create(outputFileName)
		if err != nil {
			log.Printf("Failed to create %s: %v", outputFileName, err)
			continue
		}
		defer outputFile.Close()

		if err := tmpl.Execute(outputFile, s); err != nil {
			log.Printf("Failed to generate method for %s: %v", s.Name, err)
		}
	}

	log.Printf("Generated methods for %d structs", len(structs))
}
