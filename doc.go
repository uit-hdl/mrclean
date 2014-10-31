/*
mrclean is a library and tool to show images on the Tromsø display wall for the purpose of data cleaning.

The code is divide in a library, mrclean, and few executables that are the Mr. Clean components.
The components must start in the right order: chronicle, core, display and gesture.
All the components need a config file, default is config.json, to be properly configured.
Using different configurations for the different components will break things.
Running all the components form the same directory will, by default, use the same configuration.
The mrcrun executable find in the cmd directory is a convenience tool to run all the components in the right order.

	The mrcrun tool is still a WIP and uses some dirty timing hacks: USE AT YOUR OWN RISK!!

Different examples of config.json are in the repository. In the cmd/gesture dir there is one with some gesture configuration examples.
The config file is a key/value storage. So far there are only two element stored layers, and circles.
Layers represent the depth of the watch directory and the name of the metadata assigned by the user.
It has to be a string containing the metaata concatenated by a path separator, on unix systems "/".
Circles represents the circle gestures recognized by the gesture component. The value of the circles key object is a list of objects containig the parameter of the circles to be recognized, for example:
	{
		"clockwise": true,
		"rounds": 2,
		"layout": "Group",
		"order": "number/color/shape/name"
    }

The layout can be for now only Group or Sort.
The order has to be a combination of the layers.
*/
package mrclean
