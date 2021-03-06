package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/google/go-github/v45/github"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
)

func updateFile() {
	tok := os.Getenv("GITHUB_TOKEN")
	if tok == "" {
		log.Fatal("Environment variable GITHUB_TOKEN is not set.")
	}
	client := github.NewClient(
		oauth2.NewClient(
			context.Background(),
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok}),
		),
	)

	fmt.Print("Fetching:")
	opt := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allRepos []*github.StarredRepository
	for {
		repos, resp, err := client.Activity.ListStarred(context.Background(), "", opt)
		if err != nil {
			log.Fatal(err)
		}
		allRepos = append(allRepos, repos...)
		fmt.Printf(" %d", len(repos))
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	fmt.Println()
	j, err := json.MarshalIndent(allRepos, "", "  ")
	if err != nil {
		log.Fatalf("json marshal failed: %v", err)
	}
	if err := os.WriteFile("stars.json", j, 0644); err != nil {
		log.Fatalf("failed to save stars.json: %v", err)
	}
	log.Println("wrote stars.json successfully.")
}

func loadFile() (out []*github.StarredRepository) {
	data, err := os.ReadFile("stars.json")
	if err != nil {
		log.Fatalf("failed to read stars.json: %v", err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		log.Fatalf("failed to unmarshal: %v", err)
	}
	return
}

func search(term, lang string) {
	term = strings.ToLower(term)
	lang = strings.ToLower(lang)

	repos := loadFile()
	sel := make([]*github.Repository, 0, len(repos))
	for _, r := range repos {
		r := r.Repository
		if !strings.Contains(strings.ToLower(r.GetFullName()), term) &&
			!strings.Contains(strings.ToLower(r.GetDescription()), term) {
			continue
		}
		if lang != "" && strings.ToLower(r.GetLanguage()) != lang {
			continue
		}
		sel = append(sel, r)
	}

	sort.SliceStable(sel, func(i, j int) bool {
		return sel[i].GetStargazersCount() > sel[j].GetStargazersCount()
	})

	for _, r := range sel {
		fmt.Printf(`
%s
   %s
   %slang=%s stars=%s%d%s updated=%s%s
`, "\x1b[43;30m "+r.GetFullName()+" \x1b[0m",
			"\x1b[1;37m"+r.GetDescription()+"\x1b[0m",
			"\x1b[2;37m",
			"\x1b[0m\x1b[1;34m"+r.GetLanguage()+"\x1b[2;37m",
			"\x1b[0m\x1b[1;34m", r.GetStargazersCount(), "\x1b[2;37m",
			"\x1b[0m\x1b[1;34m"+r.GetPushedAt().Format("2-Jan-2006")+"\x1b[0m",
			"\x1b[0m",
		)
	}
	if len(sel) > 0 {
		fmt.Println()
	}
}

func main() {
	log.SetFlags(0)

	fs := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	lang := fs.StringP("lang", "l", "", "show only repos written in this language")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage: ghstars update
       ghstars [options] search search-term

Options are:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err == pflag.ErrHelp {
		os.Exit(0)
	} else if err != nil {
		os.Exit(1)
	}

	log.SetPrefix("ghstars: ")
	nargs := fs.NArg()
	if nargs == 1 && fs.Arg(0) == "update" {
		updateFile()
	} else if nargs == 2 && fs.Arg(0) == "search" {
		search(fs.Arg(1), *lang)
	} else {
		fs.Usage()
		os.Exit(1)
	}
}
