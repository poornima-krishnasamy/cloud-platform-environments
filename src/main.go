package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var root string = "/Users/pkrishnasamy/Documents/poornima-krishnasamy/cloud-platform-environments/namespaces/cp-2501-1602.cloud-platform.service.justice.gov.uk"

func kubectlApply(folder string) error {

	if filepath.Base(folder) == "resources" {
		return nil
	}

	kubectlArgs := []string{"-n", filepath.Base(folder), "apply", "-f", folder}

	kubectlCommand := exec.Command("kubectl", kubectlArgs...)
	kubectlCommand.Stderr = os.Stderr
	kubectlCommand.Stdout = os.Stdout

	if err := kubectlCommand.Run(); err != nil {
		return err
	}

	return nil
}

func terraformApply(folder string) error {

	if filepath.Base(folder) != "resources" {
		return nil
	}

	// Get the value of an Environment Variable
	bucket := os.Getenv("PIPELINE_STATE_BUCKET")
	key_prefix := os.Getenv("PIPELINE_STATE_KEY_PREFIX")
	lock_table := os.Getenv("PIPELINE_TERRAFORM_STATE_LOCK_TABLE")
	region := os.Getenv("PIPELINE_STATE_REGION")
	cluster := os.Getenv("PIPELINE_CLUSTER")

	//Checking that an environment variable is present or not.

	env_vars := [3]string{
		"TF_VAR_cluster_name",
		"TF_VAR_cluster_state_bucket",
		"TF_VAR_cluster_state_key"}

	for _, env_var := range env_vars {

		// `os.Getenv` cannot differentiate between an explicitly set empty value
		// and an unset value. `os.LookupEnv` is preferred to `syscall.Getenv`,
		// but it is only available in go1.5 or newer. We're using Go build tags
		// here to use os.LookupEnv for >=go1.5
		_, ok := os.LookupEnv(env_var)
		if !ok {
			fmt.Printf("required key %s missing value", env_var)
			return nil
		}

		continue
	}

	key := key_prefix + cluster + "/" + filepath.Base(filepath.Dir(folder)) + "/terraform.tfstate"

	if err := os.Chdir(folder); err != nil {
		fmt.Println(err)
		return err
	}

	kubectlArgs := []string{
		"init",
		fmt.Sprintf("%s=bucket=%s", "-backend-config", bucket),
		fmt.Sprintf("%s=key=%s", "-backend-config", key),
		fmt.Sprintf("%s=dynamodb_table=%s", "-backend-config", lock_table),
		fmt.Sprintf("%s=region=%s", "-backend-config", region)}

	kubectlCommand := exec.Command("terraform", kubectlArgs...)
	kubectlCommand.Stderr = os.Stderr
	kubectlCommand.Stdout = os.Stdout

	if err := kubectlCommand.Run(); err != nil {
		fmt.Println(err)
		return err
	}

	if err := os.Chdir(root); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func listFiles(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.IsDir() {
			fmt.Printf("skipping a dir without errors: %+v \n", info.Name())
			return filepath.SkipDir
		}
		*files = append(*files, path)
		return nil
	}
}

func listFolders(folders *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.Name() == ".terraform" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			*folders = append(*folders, path)
		}
		return nil
	}
}

func applyNamespaceDir(folder string) (tferror string, resp string) {

	var outb, errb bytes.Buffer

	if filepath.Base(folder) != "resources" {

		kubectlArgs := []string{"-n", filepath.Base(folder), "apply", "-f", folder}

		kubectlCommand := exec.Command("kubectl", kubectlArgs...)

		kubectlCommand.Stdout = &outb
		kubectlCommand.Stderr = &errb
		kubectlCommand.Run()

	}

	if filepath.Base(folder) == "resources" {

		// Get the value of an Environment Variable
		bucket := os.Getenv("PIPELINE_STATE_BUCKET")
		key_prefix := os.Getenv("PIPELINE_STATE_KEY_PREFIX")
		lock_table := os.Getenv("PIPELINE_TERRAFORM_STATE_LOCK_TABLE")
		region := os.Getenv("PIPELINE_STATE_REGION")
		cluster := os.Getenv("PIPELINE_CLUSTER")

		//Checking that an environment variable is present or not.

		env_vars := [3]string{
			"TF_VAR_cluster_name",
			"TF_VAR_cluster_state_bucket",
			"TF_VAR_cluster_state_key"}

		for _, env_var := range env_vars {

			// `os.Getenv` cannot differentiate between an explicitly set empty value
			// and an unset value. `os.LookupEnv` is preferred to `syscall.Getenv`,
			// but it is only available in go1.5 or newer. We're using Go build tags
			// here to use os.LookupEnv for >=go1.5
			_, ok := os.LookupEnv(env_var)
			if !ok {
				return
			}

			continue
		}

		key := key_prefix + cluster + "/" + filepath.Base(filepath.Dir(folder)) + "/terraform.tfstate"

		os.Chdir(folder)

		kubectlArgs := []string{
			"init",
			fmt.Sprintf("%s=bucket=%s", "-backend-config", bucket),
			fmt.Sprintf("%s=key=%s", "-backend-config", key),
			fmt.Sprintf("%s=dynamodb_table=%s", "-backend-config", lock_table),
			fmt.Sprintf("%s=region=%s", "-backend-config", region)}

		Command := exec.Command("terraform", kubectlArgs...)

		Command.Stdout = &outb
		Command.Stderr = &errb
		Command.Run()

		os.Chdir(root)
	}

	return errb.String(), outb.String()
}

func main() {

	type Result struct {
		Error    string
		Response string
	}
	checkStatus := func(done <-chan interface{}, folders ...string) <-chan Result {
		results := make(chan Result)
		go func() {
			defer close(results)

			for _, folder := range folders {
				var result Result
				err, resp := applyNamespaceDir(folder)
				result = Result{Error: err, Response: resp}
				select {
				case <-done:
					return
				case results <- result:
				}
			}
		}()
		return results
	}
	done := make(chan interface{})
	defer close(done)

	var folders []string

	err := filepath.Walk(root, listFolders(&folders))
	if err != nil {
		panic(err)
	}
	for result := range checkStatus(done, folders...) {
		if result.Error != "" {
			fmt.Printf("error: %v", result.Error)
			continue
		}
		fmt.Printf("Response: %v\n", result.Response)
	}

}
