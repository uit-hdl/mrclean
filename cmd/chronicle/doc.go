/*
chronicle is the chronicle component of mrclean

The chronicle component watches the directory tree where the images and visualization code reside.
On a change of the visualization code or on a new image being put in the tree, chronicle will
commit it to a git repository created on startup.
The commit message is the timestamp and the session name. The session name is modifiable with a
command line parameter (see chronicle -help).
At startup if another git repo is present chronicle will not work properly, so be sure to delete or
move any repo present in the watch directory. The watch directory is modifiable from command line at
startup.
On a new image the chronicle component sends a RPC call to the core component containing the name of the new image, the size and
the URL. DisplayCloud uses the URL to fetch the image and display it. To this end chronicle also runs an HTTP server, serving the images in the watch directory tree.

chronicles supports png and jpeg image files so far.


*/
package main
