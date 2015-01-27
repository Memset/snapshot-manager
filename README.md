Snapshot Manager
================

Snapshot Manager is a command line program to manage Memset Miniserver snapshots.

See the [Memset snapshot
documentation](http://www.memset.com/docs/managing-your-server/server-snapshots/)
for how to create a snapshot of your Miniserver.  This tool will
enable you to download the snapshots, delete old ones and upload new
ones you can create new virtual machines from.

Features

  * List snapshots
  * Upload a new snapshot that you can create Miniservers from
  * Download an existing snapshot
  * Delete an existing snapshot

Install
-------

Snapshot Manager is a Go program and comes as a single binary file.

Download the binary for your OS from

  * https://github.com/Memset/snapshot-manager/releases/latest

Or alternatively if you have Go installed use

    go install github.com/memset/snapshot-manager

and this will build the binary in `$GOPATH/bin`.

Configure
---------

At minimum snapshot-manager needs a username and password for the Memstore it operates on.

These can be supplied as command line arguments like this

    snapshot-manager -user myaccaa1.admin -password eVyjCyp4 list

Or more conveniently they can be stored in a config file in your home directory.

    snapshot-manager -h

Will show the default location of the config file for your OS - see
the `-config` section.  This should be a file called
`.snapshot-manager.conf` in your home directory.  Eg

    -config="/home/user/.snapshot-manager.conf": Path to config file

Edit this file and put your user and password in like this:

    user = "myaccaa1.admin"
    password = "eVyjCyp4"

You can then use snapshot-manager without the `-user` and `-password` parameters, eg

    snapshot-manager list

The examples will show this form.

It is recommended that you make a user which can only access the
`miniserver-snapshots` container for use with snapshot-manager.  See
[the Memset Memstore
documentation](http://www.memset.com/docs/other-memset-services/memstore/container-access-control-list/)
for how to do that.

Usage
-----

Run snapshot-manager without any parameters to see the help
```
snapshot-manager version v1.00 (C) Memset Ltd 2015

Manage snapshots in Memset Memstore

snapshot-manager <command> <arguments>

Commands

  list             - lists the snapshots
  download name    - downloads the snapshot
  upload name file - uploads a disk image as a snapshot
  delete name      - deletes the snapshot
  types            - available snapshot types

Full options:
  -auth-url="https://auth.storage.memset.com/v1.0": Swift Auth URL - default is for Memstore
  -chunk-size=67108864: Size of the chunks to make
  -config="/home/user/.snapshot-manager.conf": Path to config file
  -password="": Memstore password
  -user="": Memstore user name, eg myaccaa1.admin
```

Options can also be stored in the config file.  The config file is in
[toml format](https://github.com/toml-lang/toml). Any options passed
in on the command line will override those from the config file.

  * `-user` can be stored in the config file as `user = "string"`
  * `-password` can be stored in the config file as `password = "string"`
  * `-auth-url` can be stored in the config file as `authurl = "string"`
  * `-s` can be stored in the config file as `chunksize = number`

You can then use the sub commands to manage your snapshots.

List
----

To list your snapshots run the list command

    snapshot-manager list

This will list something like this showing the snapshots and some
details about them.  The following commands take a snapshot name which
is on the first line of each section, eg `myacc.2012-08-21-17-08-27`.

```
myacc.2012-08-21-17-08-27
  Comment    - A real snapshot
  Path       - myacc.2012-08-21-17-08-27/myacc1.tar
  Date       - 2012-08-21 17:08:27.391967 +0000 UTC
  Broken     - false
  Miniserver - myacc1
  ImageType  - Tarball file
  ImageLeaf  - myacc1.tar
  Md5        - 3350b64f1b48cc2f3a10d6fda6b18b43
myacc.2015-01-08-15-44-16
  Comment    - Another real snapshot
  Path       - myacc.2015-01-08-15-44-16/myacc1.tar
  Date       - 2015-01-08 15:44:16.695676 +0000 UTC
  Broken     - false
  Miniserver - myacc1
  ImageType  - Tarball file
  ImageLeaf  - myacc1.tar
  Md5        - 09e29a798ec4f3e4273981cc176adc32
  DiskSize   - 42949672960
```

Download
--------

To download a snapshot use the download command.  This will create a
directory with the name of the snapshot and download it there.  Note
that this can take quite a long time as snapshots can be large.

    snapshot-manager download snapshot-name

Eg

```
$ /snapshot-manager download myacc.2015-01-08-15-44-16
Downloading myacc.2015-01-08-15-44-16/README.txt
Downloading myacc.2015-01-08-15-44-16/myacc1.tar
```

Upload
------

To upload a snapshot first prepare an image.  See the Types section
below for acceptable image types.

    snapshot-manager upload snapshot-name /path/to/snapshot/file

You can then use this image in the web interface or the API to
re-image a server, or to setup a new server.

Eg

```
$ /snapshot-manager upload new_image new_image.tar
2015/01/11 12:28:12 Uploading snapshot
2015/01/11 12:28:12 Uploading chunk "new_image/new_image.part/0001"
2015/01/11 12:28:12 Uploading chunk "new_image/new_image.part/0002"
2015/01/11 12:28:13 Uploading chunk "new_image/new_image.part/0003"
2015/01/11 12:28:13 Uploading chunk "new_image/new_image.part/0004"
...
2015/01/11 12:30:11 Uploading chunk "new_image/new_image.part/0381"
2015/01/11 12:30:11 Uploading chunk "new_image/new_image.part/0382"
2015/01/11 12:30:11 Uploading chunk "new_image/new_image.part/0383"
2015/01/11 12:30:11 Uploading chunk "new_image/new_image.part/0384"
2015/01/11 12:30:11 Uploading manifest "new_image/new_image.tar"
```

Delete
------

To delete a snapshot use the delete command.

    snapshot-manager delete snapshot-name

Eg

```
$ /snapshot-manager delete new_image
2015/01/11 12:33:39 Deleting "new_image/README.txt"
2015/01/11 12:33:39 Deleting "new_image/new_image.tar"
2015/01/11 12:33:39 Deleting "new_image/new_image.part/0001"
2015/01/11 12:33:39 Deleting "new_image/new_image.part/0002"
2015/01/11 12:33:39 Deleting "new_image/new_image.part/0003"
2015/01/11 12:33:39 Deleting "new_image/new_image.part/0004"
...
2015/01/11 12:34:41 Deleting "new_image/new_image.part/0381"
2015/01/11 12:34:41 Deleting "new_image/new_image.part/0382"
2015/01/11 12:34:41 Deleting "new_image/new_image.part/0383"
2015/01/11 12:34:41 Deleting "new_image/new_image.part/0384"
```

Types
-----

You can list information about the types of snapshots that snapshot
manager can deal with like this.

    snapshot-manager types

Note carefully the different virtualization types for each image.  Not
all image formats can be uploaded.

`raw` or `raw.gz` are the recommended formats for full virtualization
uploads.  These are supported by all virtualization systems, but may
need some conversion.  Then can also be quite large - see [How to
improve performance for raw
snapshots](http://www.memset.com/blog/improving-raw-snapshots-performance/)
to learn how to make smaller raw images.  These are always stored
compressed.

`tar` or `tar.gz` are the recommended formats for paravirtualized Linux
uploads. If you are making one of these then we recommend you start
with a Memset image as these are customized to enable networking and
serial console to work.


```
.tar - Tarball file
  Upload:         true
  Comment:        A tar of whole file system
  Virtualisation: Paravirtualisation - Linux only
.tar.gz - Tarball file
  Upload:         true
  Comment:        A tar of whole file system
  Virtualisation: Paravirtualisation - Linux only
.raw.gz - gzipped Raw file
  Upload:         true
  Comment:        A raw disk image including partitions, gzipped
  Virtualisation: Full virtualisation with PV Drivers
.raw - gzipped Raw file
  Upload:         true
  Comment:        A raw disk image including partitions
  Virtualisation: Full virtualisation with PV Drivers
.xmbr - gzipped NTFS Image file
  Upload:         false
  Comment:        ntfsclone + boot sector + partitions
  Virtualisation: Full virtualisation with PV Drivers
.vmdk - VMDK
  Upload:         false
  Comment:        Raw disk image with partitions, VMDK format
  Virtualisation: Full virtualisation with PV Drivers
.vhd - VHD
  Upload:         false
  Comment:        Raw disk image with partitions, VMDK format
  Virtualisation: Full virtualisation with PV Drivers
```

License
-------

This is free software under the terms of MIT the license (check the
COPYING file included in this package).

Changelog
---------
  * v1.00 - 2015-01-26

Contact and support
-------------------

The project website is at:

  * https://github.com/memset/snapshot-manager

There you can file bug reports, ask for help or send pull requests.

Authors
-------

  * Nick Craig-Wood <nick@memset.com>

Contributors
------------

  * Your name goes here!
