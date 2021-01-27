package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// func runKubectl(args []string) error {
// 	args = append(args, fmt.Sprintf("--namespace=%s", namespace))

// 	cmd := exec.Command("kubectl", args...)
// 	cmd.Stderr = os.Stderr
// 	cmd.Stdout = os.Stdout

// 	return cmd.Run()
// }

func kubectlapply(file string) error {

	var rawResources [][]byte

	var manifest io.Reader

	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		resp, err := http.Get(file)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("unable to read URL %s, server reported %d", file, resp.StatusCode)
		}

		defer resp.Body.Close()
		manifest = resp.Body

	} else if file == "-" {
		manifest = os.Stdin
	} else {
		var err error
		manifest, err = os.Open(file)
		if err != nil {
			return err
		}
	}

	decoder := k8sYaml.NewYAMLOrJSONDecoder(manifest, 4096)

	var obj *unstructured.Unstructured

	for {
		err := decoder.Decode(&obj)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to unmarshal manifest: %s", err)
		}

		if obj == nil {
			break
		}

		var resource interface{}

		resource = obj

		rawResource, err := yaml.Marshal(resource)
		if err != nil {
			return err
		}

		rawResources = append(rawResources, rawResource)

		obj = nil
	}

	resourceBuffer := bytes.NewBuffer(nil)

	for _, rawResource := range rawResources {
		_, err := resourceBuffer.Write(rawResource)
		if err != nil {
			return err
		}

		_, err = resourceBuffer.WriteString("\n---\n")
		if err != nil {
			return err
		}
	}

	kubectlArgs := []string{"apply", "-f", "-"}

	kubectlCommand := exec.Command("kubectl", kubectlArgs...)
	kubectlCommand.Stdin = resourceBuffer
	kubectlCommand.Stderr = os.Stderr
	kubectlCommand.Stdout = os.Stdout

	if err := kubectlCommand.Run(); err != nil {
		return err
	}

	return nil
}

func listfiles(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if filepath.Ext(path) == ".yaml" {
			*files = append(*files, path)
		}
		return nil
	}
}

func listfolders(folders *[]string) filepath.WalkFunc {
	subDirToSkip := "resources"
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.IsDir() && info.Name() == subDirToSkip {
			return filepath.SkipDir
		}

		if info.IsDir() {
			*folders = append(*folders, path)
		}
		return nil
	}
}

func main() {

	var folders []string

	root := "../namespaces/cp-2501-1602.cloud-platform.service.justice.gov.uk"
	err := filepath.Walk(root, listfolders(&folders))
	if err != nil {
		panic(err)
	}

	for _, folder := range folders {

		var files []string

		root := folder

		fmt.Println(folder)
		err := filepath.Walk(root, listfiles(&files))
		if err != nil {
			panic(err)
		}
		for _, file := range files {
			kubectlapply(file)
		}
	}
}
