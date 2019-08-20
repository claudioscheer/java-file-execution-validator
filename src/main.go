package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/macaron.v1"
)

var expectedResult = "Olá, mundo!"

// Only one request at a time because of the file name.
// I'll not do something better than that.
var mutex = &sync.Mutex{}

func main() {
	m := macaron.Classic()
	m.Use(macaron.Renderer())

	m.Get("/", func(ctx *macaron.Context) {
		ctx.HTML(200, "upload")
	})

	m.Get("/upload", func(ctx *macaron.Context) {
		ctx.Redirect("/")
	})

	m.Post("/upload", func(ctx *macaron.Context) {
		mutex.Lock()
		fr, header, err := ctx.Req.FormFile("java-file")
		if err != nil {
			responseHelper(ctx, err)
			return
		}
		defer fr.Close()

		if !fileNameValidator(header.Filename) {
			responseHelper(ctx, errors.New("o nome do arquivo é inválido"))
			return
		}

		extension := filepath.Ext(header.Filename)
		fileName := strings.TrimSuffix(header.Filename, extension)

		filePath := path.Join("../uploads", header.Filename)

		fw, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			responseHelper(ctx, err)
			return
		}
		defer fw.Close()

		_, err = io.Copy(fw, fr)
		if err != nil {
			responseHelper(ctx, err)
			return
		}

		compiled := compileJavaFile(header.Filename)
		if compiled {
			responseHelper(ctx, errors.New("erro ao compilar"))
			return
		}

		compiledResult := runJavaCompiled(fileName)
		if compiledResult == expectedResult {
			responseHelper(ctx, "OK. Algoritmo funcionando.")
		} else {
			result := fmt.Sprintf("o resultado esperado é '%s', porém, o resultado real é '%s'", expectedResult, compiledResult)
			responseHelper(ctx, result)
		}
	})

	m.Run()
}

func responseHelper(ctx *macaron.Context, msg interface{}) {
	ctx.Data["Mensagem"] = msg
	ctx.HTML(200, "upload")
	mutex.Unlock()
}

func compileJavaFile(fileName string) bool {
	result, err := exec.Command("/bin/bash", "../scripts/compile.sh", fileName).Output()
	if err != nil {
		return false
	}
	return string(result) == "Compiled."
}

func runJavaCompiled(fileName string) string {
	result, err := exec.Command("/bin/bash", "../scripts/run-compiled.sh", fileName).Output()
	if err != nil {
		return fmt.Sprint(err)
	}
	return strings.TrimSpace(string(result))
}

func fileNameValidator(fileName string) bool {
	validFileName := regexp.MustCompile(`^([a-zA-Z0-9]*)\.java$`)
	return validFileName.MatchString(fileName)
}
