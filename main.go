package main

import (
	"io"
	"net/url"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"github.com/jhunt/go-s3"
)

var opts struct {
	Help bool `cli:"-h, --help"`

	ID     string `cli:"--aki"        env:"S3_AKI"`
	Key    string `cli:"--key"        env:"S3_KEY"`
	URL    string `cli:"--s3-url"     env:"S3_URL"`
	Region string `cli:"-r, --region" env:"S3_REGION"`

	CreateBucket struct {
		ACL string `cli:"--acl"`
	} `cli:"create-bucket"`

	DeleteBucket struct {
		Recursive bool `cli:"-R"`
	} `cli:"delete-bucket"`

	Bucket string `cli:"-b, --bucket" env:"S3_BUCKET"`

	Upload struct {
		To string `cli:"--to"`
	} `cli:"upload"`

	Cat struct {
	} `cli:"cat, get"`

	Delete struct {
	} `cli:"rm, delete"`

	List struct {
	} `cli:"list"`
}

func client() (*s3.Client, error) {
	domain := ""
	if opts.URL != "" {
		u, err := url.Parse(opts.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid --s3-url '%s': %s", opts.URL, err)
		}
		domain = u.Host
	}

	if opts.ID == "" {
		return nil, fmt.Errorf("missing required --id (or $S3_AKI) value")
	}

	if opts.Key == "" {
		return nil, fmt.Errorf("missing required --key (or $S3_KEY) value")
	}

	return s3.NewClient(&s3.Client{
		AccessKeyID:     opts.ID,
		SecretAccessKey: opts.Key,
		Domain:          domain,
		Region:          opts.Region,
		Bucket:          opts.Bucket,
	})
}

func bail(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!! %s}\n", err)
		os.Exit(2)
	}
}

func main() {
	env.Override(&opts)
	opts.Region = "us-east-1"
	opts.CreateBucket.ACL = "private"

	command, args, err := cli.Parse(&opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!! %s}\n", err)
		os.Exit(1)
	}

	if command == "create-bucket" {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing bucket name argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 create-bucket NAME\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 create-bucket NAME\n")
			os.Exit(1)
		}

		c, err := client()
		bail(err)

		c.Region = "us-east-1"
		err = c.CreateBucket(args[0], "", opts.CreateBucket.ACL)
		bail(err)

		fmt.Printf("bucket @Y{%s} created.\n", args[0])
		os.Exit(0)
	}

	if command == "delete-bucket" {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing bucket name argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 delete-bucket NAME\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 delete-bucket NAME\n")
			os.Exit(1)
		}

		c, err := client()
		bail(err)

		c.Region = "us-east-1"
		err = c.DeleteBucket(args[0])
		bail(err)

		fmt.Printf("bucket @Y{%s} deleted.\n", args[0])
		os.Exit(0)
	}

	if command == "upload" {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing bucket name argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 upload PATH\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 upload PATH\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		if opts.Upload.To == "" {
			if args[0] == "-" {
				bail(fmt.Errorf("uploading from stdin requires the --to option."))
			}
			opts.Upload.To = strings.TrimLeft(args[0], "./")
		}

		c, err := client()
		bail(err)

		u, err := c.NewUpload(opts.Upload.To)
		bail(err)

		from := os.Stdin
		if args[0] != "-" {
			from, err = os.Open(args[0])
			bail(err)
			defer from.Close()
		}

		_, err = u.Stream(from, 5*(2<<30))
		bail(err)

		err = u.Done()
		bail(err)

		os.Exit(0)
	}

	if command == "cat" {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 cat PATH\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 cat PATH\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		c, err := client()
		bail(err)

		out, err := c.Get(args[0])
		bail(err)

		_, err = io.Copy(os.Stdout, out)
		bail(err)

		os.Exit(0)
	}

	if command == "rm" {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 rm PATH\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 rm PATH\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		c, err := client()
		bail(err)

		err = c.Delete(args[0])
		bail(err)

		os.Exit(0)
	}

	if command == "list" {
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: s3 list [-b BUCKET]\n")
			os.Exit(1)
		}

		c, err := client()
		bail(err)

		c.Bucket = opts.Bucket
		files, err := c.List()
		bail(err)

		for _, f := range files {
			fmt.Printf("- %s %s %s %s %s\n", f.Key, f.LastModified, f.OwnerName, f.ETag, f.Size)
		}
		os.Exit(0)
	}
}
