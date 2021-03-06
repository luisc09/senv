package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

var (
	client ssmiface.SSMAPI
)

func init() {
	session := session.Must(session.NewSession())
	client = ssm.New(session)
}

var parameters map[string]string

type parameter struct {
	name  string
	value string
}

func main() {
	paths := flag.String("paths", "/", "Comma separated paths to look for, i.e. /dev/service, /dev/global/")
	export := flag.Bool("export", false, "If present, it will add the the export string to the variable")
	flag.Parse()
	parameters, err := fetchParametersByPaths(splitPaths(*paths))
	if err != nil {
		panic(err)
	}
	if len(parameters) != 0 {
		fmt.Print(formatParameters(parameters, *export))
	} else {
		panic("Parameters not found")
	}

}

func splitPaths(paths string) []string {
	return strings.Split(paths, ",")
}

func fetchParametersByPaths(paths []string) (map[string]string, error) {
	parameters = make(map[string]string)
	for _, path := range paths {
		p, err := fetchParametersByPath(path)
		if err != nil {
			return map[string]string{}, err
		}
		for k, v := range p {
 		   parameters[k] = v
		}
	}
	return parameters, nil
}

func fetchParametersByPath(path string) (map[string]string, error) {
	var parameters = make(map[string]string)

	done := false
	var token string
	for !done {
		input := &ssm.GetParametersByPathInput{
			Path:           &path,
			WithDecryption: aws.Bool(true),
		}
		if token != "" {
			input.SetNextToken(token)
		}
		output, err := client.GetParametersByPath(input)
		if err != nil {
			return map[string]string{}, err
		}
		for _, p := range output.Parameters {
			name := *p.Name
			value := *p.Value
			if strings.Compare(path, "/") != 0 {
				name = strings.Replace(strings.Trim(name[len(path):], "/"), "/", "_", -1)
			}
			parameters[name] = value
		}
		if output.NextToken != nil {
			token = *output.NextToken
		} else {
			done = true
		}
	}
	return parameters, nil
}

func formatParameters(parameters map[string]string, export bool) string {
	var buffer strings.Builder
	var prefix string
	if export {
		prefix = "export "
	}
	for key, value := range parameters {
		buffer.WriteString(fmt.Sprintf("%s%s=%s\n", prefix, key, value))
	}
	return buffer.String()
}
