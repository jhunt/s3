package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"github.com/jhunt/go-s3"
)

var Version = ""

var opts struct {
	Help    bool `cli:"-h, --help"`
	Version bool `cli:"-v, --version"`
	Debug   bool `cli:"-D, --debug"   env:"S3_DEBUG"`
	Trace   bool `cli:"-T, --trace"   env:"S3_TRACE"`

	ID     string `cli:"--aki"        env:"S3_AKI"`
	Key    string `cli:"--key"        env:"S3_KEY"`
	URL    string `cli:"--s3-url"     env:"S3_URL"`
	Region string `cli:"-r, --region" env:"S3_REGION"`

	SkipVerify bool `cli:"-k, --insecure"     env:"S3_INSECURE"`
	PathBased  bool `cli:"-P, --path-buckets" env:"S3_USE_PATH"`

	Recursive bool `cli:"-R"`

	Commands struct{} `cli:"commands"`
	ACLs     struct{} `cli:"acls"`

	ShowHelp struct{} `cli:"help"`

	ListBuckets struct {
	} `cli:"list-buckets, lsb"`

	CreateBucket struct {
		ACL string `cli:"--acl, --policy" env:"S3_ACL"`
	} `cli:"create-bucket, new-bucket, cb"`

	DeleteBucket struct {
	} `cli:"delete-bucket, remove-bucket"`

	Bucket string `cli:"-b, --bucket" env:"S3_BUCKET"`

	Upload struct {
		To          string `cli:"--to"`
		ContentType string `cli:"-t, --content-type"`
		Parallel    int    `cli:"-n, --parallel"      env:"S3_THREADS"`
	} `cli:"put, upload"`

	Download struct {
		To string `cli:"--to"`
	} `cli:"get, download"`

	Cat struct {
	} `cli:"cat"`

	GenerateURL struct {
	} `cli:"url"`

	Delete struct {
	} `cli:"rm, remove, delete"`

	List struct {
	} `cli:"ls, list"`

	ChangeACL struct {
	} `cli:"chacl, change-acl"`

	ListACL struct {
	} `cli:"lsacl, list-acl"`
}

func client() (*s3.Client, error) {
	domain := ""
	scheme := ""
	if opts.URL != "" {
		debugf("parsing domain from url @G{%s}...", opts.URL)
		u, err := url.Parse(opts.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid --s3-url '%s': %s", opts.URL, err)
		}
		domain = u.Host
		scheme = u.Scheme
	}

	if opts.ID == "" {
		return nil, fmt.Errorf("missing required --id (or $S3_AKI) value")
	}
	debugf("setting AKI to @G{%s}", opts.ID)

	if opts.Key == "" {
		return nil, fmt.Errorf("missing required --key (or $S3_KEY) value")
	}
	debugf("setting Key to @G{%s}", opts.Key)
	if opts.Bucket != "" {
		debugf("using bucket @G{%s} in region @G{%s}", opts.Bucket, opts.Region)
	} else if opts.Region != "" {
		debugf("operating in region @G{%s}", opts.Region)
	} else {
		debugf("no @B{bucket} or @B{region} set")
	}

	if opts.Trace {
		os.Setenv("S3_TRACE", "yes")
	}

	return s3.NewClient(&s3.Client{
		AccessKeyID:        opts.ID,
		SecretAccessKey:    opts.Key,
		Domain:             domain,
		Protocol:           scheme,
		Region:             opts.Region,
		Bucket:             opts.Bucket,
		UsePathBuckets:     opts.PathBased,
		InsecureSkipVerify: opts.SkipVerify,
	})
}

func bail(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!! %s}\n", err)
		os.Exit(2)
	}
}

func debugf(m string, args ...interface{}) {
	if opts.Debug {
		fmt.Fprintf(os.Stderr, "@Y{DEBUG> }"+m+"\n", args...)
	}
}

func version() string {
	if Version == "" {
		return "(development version)"
	} else {
		return fmt.Sprintf("v%s", Version)
	}
}

func main() {
	env.Override(&opts)
	opts.Region = "us-east-1"
	opts.CreateBucket.ACL = "private"
	opts.Upload.Parallel = 2

	command, args, err := cli.Parse(&opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!! %s}\n", err)
		os.Exit(1)
	}

	if opts.Version {
		fmt.Printf("s3 %s\n", version())
		os.Exit(0)
	}

	if command == "help" || (opts.Help && command == "") {
		fmt.Printf("General usage: @G{s3} @C{COMMAND} @W{[OPTIONS...]}\n\n")
		fmt.Printf("OPTIONS\n\n")
		fmt.Printf("  --help, -h      Show this help screen.\n")
		fmt.Printf("  --version, -v   Print s3 version information, then exit.\n")
		fmt.Printf("  --debug, -D     Enable verbose logging of what s3 is doing.\n")
		fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n")
		fmt.Printf("\n")
		fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
		fmt.Printf("                  the $S3_AKI environment variable.\n")
		fmt.Printf("\n")
		fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
		fmt.Printf("                  via the $S3_KEY environment variable.\n")
		fmt.Printf("\n")
		fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
		fmt.Printf("                  should be suitable for actual AWS S3.\n")
		fmt.Printf("                  Can be set via $S3_URL.\n")
		fmt.Printf("\n")
		fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
		fmt.Printf("                  Can be set via $S3_REGION.\n")
		fmt.Printf("\n")
		fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
		fmt.Printf("  -P              By default, s3 uses DNS (name) based bucket\n")
		fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
		fmt.Printf("                  Can be set via $S3_USE_PATH=yes.\n")
		fmt.Printf("\n")
		fmt.Printf("For a list of all available s3 commands, run `@W{s3 commands}'\n")
		os.Exit(0)
	}

	debugf("@G{s3} %s starting up...", version())
	debugf("determined command to be '@C{%s}'", command)
	debugf("determined arguments to be @C{%v}", args)

	if command == "commands" {
		fmt.Printf("General usage: @G{s3} @C{COMMAND} @W{[OPTIONS...]}\n\n")
		fmt.Printf("  @C{acls}            List known ACLs and their purposes / access rules.\n")
		fmt.Printf("  @C{commands}        List known sub-commands of this s3 client.\n")
		fmt.Printf("\n")
		fmt.Printf("  @C{list-buckets}    List all S3 buckets owned by you.\n")
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
		fmt.Printf("  @C{chacl}           Change the ACL on a bucket or a file.\n")
		fmt.Printf("  @C{lsacl}           List the ACL on a bucket or a file.\n")
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

	if command == "list-buckets" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{list-buckets} [OPTIONS]\n")
			fmt.Printf("@M{List all buckets you own in S3}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

			os.Exit(0)
		}
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{list-buckets} [OPTIONS]\n")
			os.Exit(1)
		}

		c, err := client()
		bail(err)

		c.Region = "us-east-1"
		debugf("listing buckets in region @G{%s}", c.Region)
		bb, err := c.ListBuckets()
		bail(err)

		if len(bb) == 0 {
			fmt.Fprintf(os.Stderr, "@R{no buckets found.}\n")
			os.Exit(0)
		}

		w := struct {
			Name         int
			CreationDate int
			OwnerName    int
		}{
			Name:         len("bucket"),
			CreationDate: len("created at"),
			OwnerName:    len("owner"),
		}
		for _, b := range bb {
			w.Name = max(w.Name, len(b.Name))
			w.CreationDate = max(w.CreationDate, len(fmt.Sprintf("%s", b.CreationDate)))
			w.OwnerName = max(w.OwnerName, len(b.OwnerName))
		}
		fmt.Printf("%-*s  %-*s  %-*s\n", w.Name, "bucket", w.CreationDate, "created at", w.OwnerName, "owner")
		for _, b := range bb {
			fmt.Printf("@G{%-*s}  %-*s  @M{%-*s}\n", w.Name, b.Name, w.CreationDate, b.CreationDate, w.OwnerName, b.OwnerName)
		}
		os.Exit(0)
	}

	if command == "create-bucket" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{create-bucket} [OPTIONS] @Y{NAME}\n")
			fmt.Printf("@M{Creates a new bucket in S3}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

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
		debugf("creating bucket @G{%s} in region @G{%s}", args[0], c.Region)
		debugf("using bucket access control policy @G{%s}", opts.CreateBucket.ACL)
		err = c.CreateBucket(args[0], "", opts.CreateBucket.ACL)
		bail(err)

		fmt.Printf("bucket @Y{%s} created with acl @C{%s}.\n", args[0], opts.CreateBucket.ACL)
		os.Exit(0)
	}

	if command == "delete-bucket" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{delete-bucket} [OPTIONS] @Y{NAME}\n")
			fmt.Printf("@M{Deletes a bucket from S3}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

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
		c.Region = "us-east-1"
		bail(err)

		if opts.Recursive {
			debugf("recursively deleting all files in bucket...")
			c.Bucket = args[0]
			files, err := c.List()
			bail(err)

			for _, f := range files {
				debugf("  - deleting @R{%s}", f.Key)
				bail(c.Delete(f.Key))
			}
		}

		debugf("deleting bucket @R{%s} from region @R{%s}", c.Bucket, c.Region)
		bail(c.DeleteBucket(args[0]))

		fmt.Printf("bucket @Y{%s} deleted.\n", args[0])
		os.Exit(0)
	}

	if command == "put" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{put} [OPTIONS] @Y{local/file/path}\n")
			fmt.Printf("@M{Uploads a local file to an S3 bucket}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to upload to.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			fmt.Printf("  --parallel N    How many parallel I/O threads to spin up for upload\n")
			fmt.Printf("  -n N            purposes.  More threads may reduce total time to\n")
			fmt.Printf("                  upload data (bandwidth permitting) at the expense\n")
			fmt.Printf("                  of increased local system CPU / RAM usage.\n")
			fmt.Printf("                  Defaults to 2.\n")
			fmt.Printf("                  Can be set via @W{$S3_THREADS=N}.\n\n")

			fmt.Printf("  --to rel/path   The relative path (inside the bucket) to upload\n")
			fmt.Printf("                  the file to.  Defaults to the given path with\n")
			fmt.Printf("                  all leading . and / characters removed.\n\n")

			fmt.Printf("  --content-type  TYPE\n")
			fmt.Printf("  -t TYPE\n")
			fmt.Printf("                  The MIME Content-Type to set for the uploaded file.\n")
			fmt.Printf("                  By default, this will be automatically detected\n")
			fmt.Printf("                  from the first 512 bytes of the input.\n\n")

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

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		for _, arg := range args {
			if arg == "-" && opts.Upload.To == "" {
				bail(fmt.Errorf("uploading from stdin requires the --to option."))
			}
		}

		if opts.Upload.To != "" && len(args) > 1 {
			bail(fmt.Errorf("the --to option cannot be specified with multiple uploads."))
		}

		c, err := client()
		bail(err)

		debugf("spinning up @W{%d} i/o thread(s) for uploading data.", opts.Upload.Parallel)

		preamble := make([]byte, 512)
		for _, file := range args {
			to := opts.Upload.To
			if to == "" {
				to = strings.TrimLeft(file, "./")
			}

			from := os.Stdin
			if file == "-" {
				debugf("streaming data from @G{standard input} to @Y{%s}:@C{%s}", c.Bucket, to)
			} else {
				debugf("uploading @C{%s} to @Y{%s}:@C{%s}", file, c.Bucket, to)
				from, err = os.Open(file)
				bail(err)
				defer from.Close()
			}

			n := 0
			ctype := ""
			if opts.Upload.ContentType != "" {
				ctype = opts.Upload.ContentType
			} else {
				debugf("@W{%s}: detecting content-type from first 512b", file)
				n, err = from.Read(preamble)
				bail(err)

				ctype = http.DetectContentType(preamble)
			}

			rd, wr := io.Pipe()
			go func() {
				for n > 0 {
					writ, err := wr.Write(preamble)
					bail(err)
					preamble = preamble[writ:]
					n -= writ
				}
				io.Copy(wr, from)
				wr.Close()
			}()

			debugf("@W{%s}: uploading @M{%s} file to @C{%s}", file, ctype, to)
			u, err := c.NewUpload(to, &http.Header{
				"Content-Type": []string{ctype},
			})
			bail(err)

			// 1<<20 == 2^20
			_, err = u.ParallelStream(rd, 5*(1<<20), opts.Upload.Parallel)
			bail(err)

			err = u.Done()
			bail(err)
		}

		os.Exit(0)
	}

	if command == "get" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{get} [OPTIONS] @Y{remote/file/path}\n")
			fmt.Printf("@M{Download a file from S3}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

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
			debugf("streaming @Y{%s}:@C{%s} to @G{standard output}", c.Bucket, args[0])
			_, err = io.Copy(os.Stdout, out)
			bail(err)
			os.Exit(0)
		}
		if opts.Download.To == "" {
			opts.Download.To = filepath.Base(args[0])
			if opts.Download.To == "." {
				bail(fmt.Errorf("I don't know how to handle the path '%s'", args[0]))
			}
			debugf("determined destination file path to be @C{%s}", opts.Download.To)
		}

		debugf("downloading @Y{%s}:@C{%s} to @C{%s}", c.Bucket, args[0], opts.Download.To)
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
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

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

		debugf("streaming @Y{%s}:@C{%s} to @G{standard output}", c.Bucket, args[0])
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
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to remove from.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			fmt.Printf("  -R              Recursively remove all of the files in the bucket\n")
			fmt.Printf("                  under the given path.  @R{This is dangerous}.\n\n")

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

		if opts.Recursive {
			root := strings.TrimSuffix(args[0], "/")
			debugf("recursively deleting all files under @Y{%s}:@C{%s}", c.Bucket, args[0])

			files, err := c.List()
			bail(err)

			for _, f := range files {
				if strings.HasPrefix(f.Key, root+"/") {
					debugf("  - deleting @R{%s}", f.Key)
					bail(c.Delete(f.Key))
				} else {
					debugf("  - skipping @C{%s}", f.Key)
				}
			}
		}

		debugf("deleting @Y{%s}:@C{%s}", c.Bucket, args[0])
		bail(c.Delete(args[0]))
		os.Exit(0)
	}

	if command == "ls" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{ls} [OPTIONS] -b @Y{BUCKET}\n")
			fmt.Printf("@M{Print the contents of a remote S3 file to standard output}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

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

		debugf("listing @Y{%s}:@C{*}", c.Bucket)
		files, err := c.List()
		bail(err)

		w := struct {
			Key          int
			LastModified int
			OwnerName    int
			ETag         int
			Size         int
		}{
			Key:          len("file"),
			LastModified: len("last modified"),
			OwnerName:    len("owner"),
			ETag:         len("etag"),
			Size:         len("size"),
		}
		for _, f := range files {
			w.Key = max(w.Key, len(f.Key))
			w.LastModified = max(w.LastModified, len(fmt.Sprintf("%s", f.LastModified)))
			w.OwnerName = max(w.OwnerName, len(f.OwnerName))
			w.ETag = max(w.ETag, len(f.ETag))
			w.Size = max(w.Size, len(fmt.Sprintf("%s", f.Size)))
		}
		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*s\n", w.Key, "file", w.LastModified, "last modified", w.OwnerName, "owner", w.ETag, "etag", w.Size, "size")
		for _, f := range files {
			fmt.Printf("@G{%-*s}  %-*s  @M{%-*s}  @C{%-*s}  @Y{%-*s}\n", w.Key, f.Key, w.LastModified, f.LastModified, w.OwnerName, f.OwnerName, w.ETag, f.ETag, w.Size, f.Size)
		}
		os.Exit(0)
	}

	if command == "chacl" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{chacl} [OPTIONS] [@Y{remote/file/path}] @Y{acl}\n")
			fmt.Printf("@M{Change the access control policy on a bucket or a file}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to change acls on.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			fmt.Printf("  -R              Recursively change acls of the files in the bucket\n")
			fmt.Printf("                  under the given path.  @R{This is dangerous}.\n\n")

			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{chacl} [OPTIONS] [@Y{remote/file/path}] @Y{acl}\n")
			os.Exit(1)
		}
		if len(args) > 2 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{chacl} [OPTIONS] [@Y{remote/file/path}] @Y{acl}\n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		c, err := client()
		bail(err)

		var path, acl string
		if len(args) == 1 {
			path = ""
			acl = args[0]
		} else {
			path = args[0]
			acl = args[1]
		}

		if opts.Recursive {
			root := strings.TrimSuffix(path, "/")
			debugf("recursively changing the acl of all files under @Y{%s}:@C{%s}", c.Bucket, root)

			files, err := c.List()
			bail(err)

			for _, f := range files {
				if strings.HasPrefix(f.Key, root+"/") {
					debugf("  - chacl @Y{%s} @C{%s}", f.Key, acl)
					bail(c.ChangeACL(f.Key, acl))
				} else {
					debugf("  - skipping @C{%s}", f.Key)
				}
			}
		}

		debugf("chacl @Y{%s} @C{%s}", path, acl)
		bail(c.ChangeACL(path, acl))
		os.Exit(0)
	}

	if command == "lsacl" {
		if opts.Help {
			fmt.Printf("USAGE: @C{s3} @G{lsacl} [OPTIONS] [@Y{remote/file/path}] \n")
			fmt.Printf("@M{list the access control policy on a bucket or a file}\n\n")
			fmt.Printf("OPTIONS\n\n")
			fmt.Printf("  --help, -h      Show this help screen.\n")
			fmt.Printf("  --version, -v   Print @G{s3} version information, then exit.\n")
			fmt.Printf("  --debug, -D     Enable verbose logging of what @G{s3} is doing.\n")
			fmt.Printf("  --trace, -T     Enable HTTP tracing of S3 communication.\n\n")

			fmt.Printf("  --aki KEY-ID    The Amazon Key ID to use.  Can be set via\n")
			fmt.Printf("                  the @W{$S3_AKI} environment variable.\n\n")

			fmt.Printf("  --key SECRET    The Amazon Secret Key to use.  Can be set\n")
			fmt.Printf("                  via the @W{$S3_KEY} environment variable.\n\n")

			fmt.Printf("  --s3-url URL    The full URL to your S3 system.  The default\n")
			fmt.Printf("                  should be suitable for actual AWS S3.\n")
			fmt.Printf("                  Can be set via @W{$S3_URL}.\n\n")

			fmt.Printf("  --region, -r    The S3 region to operate in.  Defaults to us-east-1.\n")
			fmt.Printf("                  Can be set via @W{$S3_REGION}.\n\n")

			fmt.Printf("  --path-buckets  Use path-based addressing for buckets.\n")
			fmt.Printf("  -P              By default, @G{s3} uses DNS (name) based bucket\n")
			fmt.Printf("                  addressing, which confuses some S3 work-alikes.\n")
			fmt.Printf("                  Can be set via @W{$S3_USE_PATH=yes}.\n\n")

			fmt.Printf("  --bucket NAME   The name of the S3 bucket to remove from.\n")
			fmt.Printf("   -b NAME        Can be set via @W{$S3_BUCKET}.\n\n")

			fmt.Printf("  -R              Recursively list acls of the files in the bucket\n")
			fmt.Printf("                  under the given path.\n\n")

			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "@R{!!! missing path argument.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{lsacl} [OPTIONS] [@Y{remote/file/path}] \n")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "@R{!!! too many arguments.}\n")
			fmt.Fprintf(os.Stderr, "USAGE: @C{s3} @G{lsacl} [OPTIONS] [@Y{remote/file/path}] \n")
			os.Exit(1)
		}

		if opts.Bucket == "" {
			bail(fmt.Errorf("missing required --bucket option."))
		}

		c, err := client()
		bail(err)

		path := ""
		if len(args) == 1 {
			path = args[0]
		}

		if opts.Recursive {
			root := strings.TrimSuffix(path, "/")
			debugf("recursively retrieving the acl of all files under @Y{%s}:@C{%s}", c.Bucket, root)

			files, err := c.List()
			bail(err)

			w := 0
			for _, f := range files {
				w = max(w, len(f.Key))
			}
			for _, f := range files {
				if root == "" || f.Key == root || strings.HasPrefix(f.Key, root+"/") {
					acl, err := c.GetACL(f.Key)
					bail(err)
					printacl(w, f.Key, acl)
				}
			}
			os.Exit(0)
		}

		acl, err := c.GetACL(path)
		bail(err)
		printacl(len(path), path, acl)
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

func printacl(width int, file string, acl s3.ACL) {
	for i, grant := range acl {
		if i == 0 {
			fmt.Printf("@Y{%*s}  ", width, file)
		} else {
			fmt.Printf("%*s  ", width, "")
		}
		if grant.GranteeName != "" {
			fmt.Printf("@M{user}  @C{%s} has @G{%s}\n", grant.GranteeName, grant.Permission)
		} else {
			fmt.Printf("@M{group} @C{%s} has @G{%s}\n", grant.Group, grant.Permission)
		}
	}

	if len(acl) == 0 {
		fmt.Printf("%s  (no grants in acl)\n", file)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
