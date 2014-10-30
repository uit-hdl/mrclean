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
*/
package mrclean
