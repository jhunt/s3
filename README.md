s3 - A Small S3 CLI
===================

`s3` is a small CLI for accessing Amazon's S3 storage.  It's
nothing too fancy; just enough tool to get the job done.

Usage
-----

Most operations need to know your Access Key ID (AKI) and Secret
Access Key (key).  Several also need to know what bucket and what
region you are interacting with.

You can pass these as CLI arguments; namely, `--aki`, `--key`,
`--bucket` or `-b`, and `--region` or `-r`.

You can also set them in your environment, by prefixing the long
option names with `S3_`, and switching everything to uppercase,
the way the gods intended environment variables to be named:

   - `S3_AKI` - Your Access Key ID.
   - `S3_KEY` - Your secret access key.
   - `S3_REGION` - The name of the AWS region.  Defaults to
     _us-east-1_, because the author lives on the east coast.
   - `S3_BUCKET` - The name of the bucket.

To create a bucket:

```
s3 create-bucket my-new-bucket
```

To create a bucket with a specific ACL:

```
s3 create-bucket my-new-bucket --acl public-read
```

(Some commonly used ACLs include `private` [the default],
`public-read`, and `public-read-write`)

To delete an empty bucket:

```
s3 delete-bucket my-old-bucket
```

To upload a file:

```
s3 put ./local/file
```

To stream a file from standard input:

```
other --program | s3 put --to where/in/s3 -
```

To list files in a bucket:

```
s3 ls
```

To delete a file:

```
s3 rm path/in/s3
```

Contributing
------------

1. Fork the repo
2. Write your code in a feature branch
3. Create a new Pull Request
