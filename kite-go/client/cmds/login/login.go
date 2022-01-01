package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	arg "github.com/alexflint/go-arg"
)

func main() {
	var args struct {
		Email    string `arg:"positional,required"`
		Password string `arg:"positional,required"`
		Verbose  bool   `arg:"-v"`
	}
	arg.MustParse(&args)

	resp, err := http.PostForm("http://localhost:46624/api/account/login", url.Values{
		"email":    []string{args.Email},
		"password": []string{args.Password},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		os.Exit(1)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("login succeeded")
	if args.Verbose {
		fmt.Println(string(buf))
	}
}
