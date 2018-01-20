package main

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
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

	Commands struct{} `cli:"commands"`
	ACLs     struct{} `cli:"acls"`

	CreateBucket struct {
		ACL string `cli:"--acl, --policy" env:"S3_ACL"`
	} `cli:"create-bucket"`

	DeleteBucket struct {
		Recursive bool `cli:"-R"`
	} `cli:"delete-bucket"`

	Bucket string `cli:"-b, --bucket" env:"S3_BUCKET"`

	Upload struct {
		To string `cli:"--to"`
	} `cli:"put,upload"`

	Download struct {
		To string `cli:"--to"`
	} `cli:"get,download"`

	Cat struct {
	} `cli:"cat"`

	GenerateURL struct {
	} `cli:"url"`

	Delete struct {
	} `cli:"rm,delete"`

	List struct {
	} `cli:"ls,list"`
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

	if command == "commands" {
		fmt.Printf("General usage: @G{s3} @C{COMMAND} @W{[OPTIONS...]}\n\n")
		fmt.Printf("  @C{acls}            List known ACLs and their purposes / access rules.\n")
		fmt.Printf("  @C{commands}        List known sub-commands of this s3 client.\n")
		fmt.Printf("\n")
		fmt.Printf("  @C{create-bucket}   Create a new bucket.\n")
		fmt.Printf("  @C{delete-bucket}   Delete an empty bucket.\n")
		fmt.Printf("\n")
		fmt.Printf("  @C{put}             Upload a new file to S3.\n")
		fmt.Printf("  @C{get}             Download a file from S3.\n")
		fmt.Printf("  @C{cat}             Print the contents of a file in S3.\n")
		fmt.Printf("  @C{url}             Print the HTTPS URL for a file in S3.\n")
		fmt.Printf("  @C{rm}              Delete file from a bucket.\n")
		fmt.Printf("  @C{ls}              List the files in a bucket.\n")
		fmt.Printf("\n")

		os.Exit(0)
	}

	if command == "acls" {
		fmt.Printf("This utility knows about the following Amazon ACLs:\n\n")
		fmt.Printf("  @C{private}\n")
		fmt.Printf("    The bucket owner will have full read/write control over the\n")
		fmt.Printf("    bucket and its constituent files.  No one else will have any\n")
		fmt.Printf("    access, whatsoever.  This is the default ACL.\n\n")

		fmt.Printf("  @C{public-read}\n")
		fmt.Printf("    The bucket owner will have full read/write control over\n")
		fmt.Printf("    everything.  Anonymous users (aka Everyone) will have read\n")
		fmt.Printf("    access to files within the bucket.\n\n")

		fmt.Printf("  @C{public-read-write}\n")
		fmt.Printf("    Like @C{public-read}, except that the Everyone group will also\n")
		fmt.Printf("    be given write access to the bucket to upload new files, overwrite\n")
		fmt.Printf("    existing files, delete files, etc.  @R{Not recommended.}\n\n")

		fmt.Printf("  @C{aws-exec-read}\n")
		fmt.Printf("    Like @C{private}, except that the Amazon EC2 system will be able\n")
		fmt.Printf("    to read files to download Amazon Machine Images (AMIs) stored in\n")
		fmt.Printf("    the bucket.  Not useful to S3 work-alike systems, generally.\n\n")

		fmt.Printf("  @C{authenticated-read}\n")
		fmt.Printf("    The bucket owner will have full read/write control over\n")
		fmt.Printf("    everything.  Authenticated users (i.e. anyone with an AWS\n")
		fmt.Printf("    account) will have read access.\n\n")

		fmt.Printf("  @C{bucket-owner-read}\n")
		fmt.Printf("    (This ACL only applies to files uploaded to buckets)\n")
		fmt.Printf("    The account who uploaded the file will have full control over it,\n")
		fmt.Printf("    but the Bucket Owner will be allowed to read it.\n\n")

		fmt.Printf("  @C{bucker-owner-full-control}\n")
		fmt.Printf("    (This ACL only applies to files uploaded to buckets)\n")
		fmt.Printf("    Both the account who uploaded the file, and the Bucket Owner, will\n")
		fmt.Printf("    have full control to the file.\n\n")

		fmt.Printf("  @C{log-delivery-write}\n")
		fmt.Printf("    The EC2 Log Delivery service will be able to create destination log\n")
		fmt.Printf("    files in this bucket and append to them.  Not generally useful to\n")
		fmt.Printf("    S3 work-alike systems.\n\n")

		os.Exit(0)
	}

	if command == "create-bucket" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{create-bucket} [OPTIONS] @Y{NAME}\n")
			fmt.Printf("@M{Creates a new bucket in S3}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --acl ACL       An ACL / policy to apply to this bucket, and\n")
			fmt.Printf("                  all files stored within.  Run `s3 acls` to see\n")
			fmt.Printf("                  a full list of defined access control lists.\n")
			fmt.Printf("                  Can be set via @W{$S3_ACL}.\n\n")
			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing bucket name argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{create-bucket} [OPTIONS] @Y{NAME}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{create-bucket} [OPTIONS] @Y{NAME}\n")
			os.Exit(1)
		}

		c, err := client()
		bail(err)

		c.Region = "us-east-1"
		err = c.CreateBucket(args[0], "", opts.CreateBucket.ACL)
		bail(err)

		fmt.Printf("bucket @Y{%s} created with acl @C{%s}.\n", args[0], opts.CreateBucket.ACL)
		os.Exit(0)
	}

	if command == "delete-bucket" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{delete-bucket} [OPTIONS] @Y{NAME}\n")
			fmt.Printf("@M{Deletes a bucket from S3}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  -R              Recursively remove all of the files in the bucket\n")
			fmt.Printf("                  before deleting it.  @R{This is dangerous}.\n\n")

			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing bucket name argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{delete-bucket} [OPTIONS] @Y{NAME}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{delete-bucket} [OPTIONS] @Y{NAME}\n")
			os.Exit(1)
		}

		c, err := client()
		bail(err)

		c.Region = "us-east-1"

		if opts.DeleteBucket.Recursive {
			c.Bucket = args[0]
			files, err := c.List()
			bail(err)

			for _, f := range files {
				bail(c.Delete(f.Key))
			}
		}

		bail(c.DeleteBucket(args[0]))
		fmt.Printf("bucket @Y{%s} deleted.\n", args[0])
		os.Exit(0)
	}

	if command == "put" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{put} [OPTIONS] @Y{local/file/path}\n")
			fmt.Printf("@M{Uploads a local file to an S3 bucket}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to upload to.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			fmt.Printf("  --to rel/path   The relative path (inside the bucket) to upload\n")
			fmt.Printf("                  the file to.  Defaults to the given path with\n")
			fmt.Printf("                  all leading . and / characters removed.\n\n")

			fmt.Printf("  You can give the file name to upload as @Y{-}, in which case\n")
			fmt.Printf("  the data to upload will be read from standard input, and the\n")
			fmt.Printf("  destination option (@W{--to}) must be specified.\n\n")
			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing bucket name argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{put} [OPTIONS] @Y{local/file/path}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{put} [OPTIONS] @Y{local/file/path}\n")
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

	if command == "get" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{get} [OPTIONS] @Y{remote/file/path}\n")
			fmt.Printf("@M{Download a file from S3}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to search.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			fmt.Printf("  --to rel/path   The relative path (locally) to download the file to.\n")
			fmt.Printf("                  Defaults to the final component of the key in the\n")
			fmt.Printf("                  bucket (i.e. a/b/c/d -> d)\n\n")

			fmt.Printf("  You can give the file name to download to as @Y{-}, in which case\n")
			fmt.Printf("  the contents of the file will be printed to standard output, which\n")
			fmt.Printf("  behaves identically to @W{s3 cat}.\n\n")
			os.Exit(0)
			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{get} [OPTIONS] @Y{remote/file/path}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{get} [OPTIONS] @Y{remote/file/path}\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		c, err := client()
		bail(err)

		out, err := c.Get(args[0])
		bail(err)

		if opts.Download.To == "-" {
			_, err = io.Copy(os.Stdout, out)
			bail(err)
			os.Exit(0)
		}
		if opts.Download.To == "" {
			opts.Download.To = filepath.Base(args[0])
			if opts.Download.To == "." {
				bail(fmt.Errorf("I don't know how to handle the path '%s'", args[0]))
			}
		}

		file, err := os.OpenFile(opts.Download.To, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		bail(err)

		_, err = io.Copy(file, out)
		bail(err)
		os.Exit(0)
	}

	if command == "cat" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{cat} [OPTIONS] @Y{remote/file/path}\n")
			fmt.Printf("@M{Print the contents of a remote S3 file to standard output}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to search.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{cat} [OPTIONS] @Y{remote/file/path}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{cat} [OPTIONS] @Y{remote/file/path}\n")
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

	if command == "url" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{url} [OPTIONS] @Y{remote/file/path}\n")
			fmt.Printf("@M{Generate an HTTPS URL for accessing a single file in a bucket}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --bucket NAME   The name of the S3 bucket that holds the file.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{url} [OPTIONS] @Y{remote/file/path}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{url} [OPTIONS] @Y{remote/file/path}\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		fmt.Printf("https://%s.s3.amazonaws.com/%s\n", opts.Bucket, args[0])
		os.Exit(0)
	}

	if command == "rm" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{rm} [OPTIONS] @Y{remote/file/path}\n")
			fmt.Printf("@M{Removes a file from an S3 bucket}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to remove from.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{rm} [OPTIONS] @Y{remote/file/path}\n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{rm} [OPTIONS] @Y{remote/file/path}\n")
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

	if command == "ls" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{ls} [OPTIONS] -b @Y{BUCKET}\n")
			fmt.Printf("@M{Print the contents of a remote S3 file to standard output}\n\n")
			fmt.Printf("OPTIONS\n")
			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to list.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			os.Exit(0)
		}
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{ls} [OPTIONS] -b @Y{BUCKET}\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
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

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "@R{!!! unrecognized command '}@Y{%s}@R{'}\n", args[0])
	} else {
		fmt.Fprintf(os.Stderr, "I have no idea what you want me to do.\n")
		fmt.Fprintf(os.Stderr, "Have you tried running @Y{s3 commands}?\n")
	}
	os.Exit(1)
}
