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

Benchmarks
----------

In the spring of 2020, `go-s3` sported a new feature that allows
parallel part uploads via separate I/O threads.  This project
picked up the `--parallel` / `-n` flags to activate that
particular logic.  This section documents performance benchmark
results performed against 10MiB, 100MiB, and 1GiB files, uploading
with a 5MiB part size.  The three bechmarks varied the number of
I/O threads across the set (1, 2, 4, 8, 16, 32).

Here are the results.

## 10MiB Upload

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `s3 put -n1 10M` | 237.7 ± 52.7 | 185.5 | 386.8 | 1.08 ± 0.27 |
| `s3 put -n2 10M` | 387.8 ± 169.5 | 188.3 | 711.1 | 1.76 ± 0.79 |
| `s3 put -n4 10M` | 263.4 ± 102.6 | 178.3 | 428.1 | 1.19 ± 0.48 |
| `s3 put -n8 10M` | 229.1 ± 54.2 | 186.6 | 401.6 | 1.04 ± 0.27 |
| `s3 put -n16 10M` | 239.0 ± 54.4 | 181.5 | 355.0 | 1.08 ± 0.28 |
| `s3 put -n32 10M` | 220.5 ± 25.1 | 190.4 | 266.1 | 1.00 |


    Summary
      's3 put -n32 10M' ran
        1.04 ± 0.27 times faster than 's3 put -n8 10M'
        1.08 ± 0.27 times faster than 's3 put -n1 10M'
        1.08 ± 0.28 times faster than 's3 put -n16 10M'
        1.19 ± 0.48 times faster than 's3 put -n4 10M'
        1.76 ± 0.79 times faster than 's3 put -n2 10M'

## 100MiB Upload

| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `s3 put -n1 100M` | 1.913 ± 0.225 | 1.643 | 2.351 | 1.02 ± 0.18 |
| `s3 put -n2 100M` | 2.017 ± 0.291 | 1.688 | 2.558 | 1.07 ± 0.21 |
| `s3 put -n4 100M` | 1.978 ± 0.250 | 1.611 | 2.342 | 1.05 ± 0.19 |
| `s3 put -n8 100M` | 1.878 ± 0.242 | 1.568 | 2.218 | 1.00 |
| `s3 put -n16 100M` | 2.119 ± 0.290 | 1.686 | 2.574 | 1.13 ± 0.21 |
| `s3 put -n32 100M` | 2.016 ± 0.338 | 1.540 | 2.660 | 1.07 ± 0.23 |

    Summary
      's3 put -n8 100M' ran
        1.02 ± 0.18 times faster than 's3 put -n1 100M'
        1.05 ± 0.19 times faster than 's3 put -n4 100M'
        1.07 ± 0.23 times faster than 's3 put -n32 100M'
        1.07 ± 0.21 times faster than 's3 put -n2 100M'
        1.13 ± 0.21 times faster than 's3 put -n16 100M'

## 1000MiB Upload

| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `s3 put -n1 1000M` | 64.794 ± 2.274 | 60.442 | 68.852 | 1.00 |
| `s3 put -n2 1000M` | 67.153 ± 2.292 | 64.442 | 70.527 | 1.04 ± 0.05 |
| `s3 put -n4 1000M` | 70.783 ± 3.312 | 66.670 | 75.328 | 1.09 ± 0.06 |
| `s3 put -n8 1000M` | 73.943 ± 3.351 | 66.887 | 77.231 | 1.14 ± 0.07 |
| `s3 put -n16 1000M` | 73.550 ± 1.172 | 71.698 | 75.985 | 1.14 ± 0.04 |
| `s3 put -n32 1000M` | 74.346 ± 8.183 | 66.683 | 91.842 | 1.15 ± 0.13 |

    Summary
      's3 put -n1 1000M' ran
        1.04 ± 0.05 times faster than 's3 put -n2 1000M'
        1.09 ± 0.06 times faster than 's3 put -n4 1000M'
        1.14 ± 0.04 times faster than 's3 put -n16 1000M'
        1.14 ± 0.07 times faster than 's3 put -n8 1000M'
        1.15 ± 0.13 times faster than 's3 put -n32 1000M'


Contributing
------------

1. Fork the repo
2. Write your code in a feature branch
3. Create a new Pull Request
