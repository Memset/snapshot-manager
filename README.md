Snapshot Manager
================

[![Logo](http://rclone.org/img/rclone-120x120.png)](http://rclone.org/)

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


FIXME
-----

Extra notes

raw for upload and download
 - How to improve performance for raw snapshots
 - http://www.memset.com/blog/improving-raw-snapshots-performance/

tar for upload and download
 - must have started with a memset image

Update snapshot types

Install
-------

Snapshot Manager is a Go program and comes as a single binary file.

Download the binary for your OS from

  * http://snapshot-manager.memset.com/

Or alternatively if you have Go installed use

    go install github.com/memset/snapshot-manager

and this will build the binary in `$GOPATH/bin`.

Configure
---------

FIXME - Config file?

FIXME instructions on where to get Memstore user name from and/or configure a limited user for Miniserver-images?

Usage
-----

Run snapshot-manager without any parameters to see the help

```
Manage snapshots in Memset Memstore

snapshot-manager <command> <arguments>

Commands

  list             - lists the snapshots
  download name    - downloads the snapshot
  upload name file - uploads a disk image as a snapshot
  delete name      - deletes the snapshot
  types            - available snapshot types

Full options:
  -api-key="": Memstore api key
  -auth-url="https://auth.storage.memset.com/v1.0": Swift Auth URL - default should be OK for Memstore
  -s=67108864: Size of the chunks to make
  -user="": Memstore user name, eg myaccaa1.admin
Flags -user and -api-key required
```

You can then use the sub commands to do things

List
----

To list your snapshots run the list command

    snapshot-manager -user USER -api-key APIKEY list

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

    snapshot-manager -user USER -api-key APIKEY download snapshot-name

Eg

```
$ /snapshot-manager -user USER -api-key APIKEY download myacc.2015-01-08-15-44-16
Downloading myacc.2015-01-08-15-44-16/README.txt
Downloading myacc.2015-01-08-15-44-16/myacc1.tar
```

Upload
------

To upload a snapshot first prepare an image.  See the Types section
below for acceptable image types.

    snapshot-manager -user USER -api-key APIKEY upload snapshot-name /path/to/snapshot/file

You can then use this image in the web interface or the API to
re-image a server, or to setup a new server.

Eg

```
$ /snapshot-manager -user USER -api-key APIKEY upload new_image new_image.tar
2015/01/11 12:28:12 Uploading snapshot
2015/01/11 12:28:12 Uploading chunk "new_image/new_image/0001"
2015/01/11 12:28:12 Uploading chunk "new_image/new_image/0002"
2015/01/11 12:28:13 Uploading chunk "new_image/new_image/0003"
2015/01/11 12:28:13 Uploading chunk "new_image/new_image/0004"
...
2015/01/11 12:30:11 Uploading chunk "new_image/new_image/0381"
2015/01/11 12:30:11 Uploading chunk "new_image/new_image/0382"
2015/01/11 12:30:11 Uploading chunk "new_image/new_image/0383"
2015/01/11 12:30:11 Uploading chunk "new_image/new_image/0384"
2015/01/11 12:30:11 Uploading manifest "new_image/new_image.tar"
```

Delete
------

To delete a snapshot use the delete command.

    snapshot-manager -user USER -api-key APIKEY delete snapshot-name

Eg

```
$ /snapshot-manager -user USER -api-key APIKEY delete new_image
2015/01/11 12:33:39 Deleting "new_image/README.txt"
2015/01/11 12:33:39 Deleting "new_image/new_image.tar"
2015/01/11 12:33:39 Deleting "new_image/new_image/0001"
2015/01/11 12:33:39 Deleting "new_image/new_image/0002"
2015/01/11 12:33:39 Deleting "new_image/new_image/0003"
2015/01/11 12:33:39 Deleting "new_image/new_image/0004"
...
2015/01/11 12:34:41 Deleting "new_image/new_image/0381"
2015/01/11 12:34:41 Deleting "new_image/new_image/0382"
2015/01/11 12:34:41 Deleting "new_image/new_image/0383"
2015/01/11 12:34:41 Deleting "new_image/new_image/0384"
```

Types
-----

You can list information about the types of snapshots that snapshot
manager can deal with like this.

    snapshot-manager -user USER -api-key APIKEY types

Note carefully the different virtualization types for each image.  Not
all image formats can be uploaded.

```
.tar - Tarball file
  Upload:         true
  Comment:        A tar of whole file system
  Virtualisation: Paravirtualisation - Linux only
.raw.gz - gzipped Raw file
  Upload:         true
  Comment:        A raw disk image including partitions, gzipped
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
.qcow2 - QCOW2
  Upload:         true
  Comment:        Raw disk image with partitions, QCOW2 format
  Virtualisation: Full virtualisation with PV Drivers
```

License
-------

This is free software under the terms of MIT the license (check the
COPYING file included in this package).

Changelog
---------
  * v1.00 - 2015-01-11

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
