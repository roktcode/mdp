package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const (
	defaultTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta http-equiv="content-type" content="text/html; charset=utf-8">
    <title>Markdown Preview Tool</title>
</head>
<body>
Previewing: {{ .FileName  }}
{{ .Body }}
</body>
</html>
`
)

// content type represents the HTML content tto add into the template
type content struct {
	Title    string
	Body     template.HTML
	FileName string
}

func main() {
	// parse flags
	filename := flag.String("file", "", "Markdown file to preview")
	skipPreview := flag.Bool("s", false, "Skip auto-preview")
	tFname := flag.String("t", "", "Alternate template name")
	flag.Parse()

	if *filename == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*filename, *tFname, os.Stdout, *skipPreview); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func run(filename string, tFname string, out io.Writer, skipPreview bool) error {
	// Read all the data from the input file and check for errors
	input, err := os.ReadFile(filename)

	if err != nil {
		return err
	}
	htmlData, err := parseContent(input, tFname, filename)

	if err != nil {
		return err
	}

	// Create temporary file and check for errors
	temp, err := ioutil.TempFile("", "mdp*.html")

	if err != nil {
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	outName := temp.Name()

	fmt.Fprintln(out, outName)

	if err := saveHTML(outName, htmlData); err != nil {
		return err
	}

	if skipPreview {
		return nil
	}

	defer os.Remove(outName)
	return preview(outName)
}

func parseContent(input []byte, tFname string, fileName string) ([]byte, error) {
	// convert markdown to html using blackfriday
	output := blackfriday.Run(input)
	// sanitize the body content against potential harmful data
	body := bluemonday.UGCPolicy().SanitizeBytes(output)

	// Parse the contents of the defaultTemplate const into a new Template
	t, err := template.New("mdp").Parse(defaultTemplate)

	if err != nil {
		return nil, err
	}

	// If user provided alternate template file, replace template
	if tFname != "" {
		t, err = template.ParseFiles(tFname)
		if err != nil {
			return nil, err
		}
	}

	// Instantiate the content type, adding the title and body
	c := content{Title: "Markdown Preview Tool", Body: template.HTML(body), FileName: fileName}

	var buffer bytes.Buffer

	// Execute the template with the content type
	if err := t.Execute(&buffer, c); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil

}

func saveHTML(outFname string, data []byte) error {
	// write the bytes to the file
	return os.WriteFile(outFname, data, 0644)
}

func preview(fname string) error {
	cName := ""
	cParams := []string{}

	// define  executable baed on OS
	switch runtime.GOOS {
	case "windows":
		cName = "cmd.exe"
		cParams = []string{"/C", "start"}
	case "linux":
		cName = "xdg-open"
	case "darwin":
		cName = "open"
	default:
		return fmt.Errorf("OS not supported")
	}
	cParams = append(cParams, fname)

	// locate executable in PATH
	cPath, err := exec.LookPath(cName)

	if err != nil {
		return err
	}
	//	open the file using the default program
	err = exec.Command(cPath, cParams...).Run()

	// give the browser some time to open the file before deleting it
	time.Sleep(2 * time.Second)

	return err
}
