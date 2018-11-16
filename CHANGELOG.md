# Changelog

## 0.0.7 - 2018-11-16

* Add logic to fix ref path if no "#" is found [zhongjiec] [[0e33a99](https://github.com/coryb/slipscheme/commit/0e33a99)]

## 0.0.6 - 2018-11-14

* Add file name for comments and removed command output [zhongjiec] [[18ed9d4](https://github.com/coryb/slipscheme/commit/18ed9d4)]

## 0.0.5 - 2018-11-13

* Remove debug prints [zhongjiec] [[331d946](https://github.com/coryb/slipscheme/commit/331d946)]
* Fix issues with definition titles [zhongjiec] [[3a3aeb7](https://github.com/coryb/slipscheme/commit/3a3aeb7)]

## 0.0.4 - 2018-11-08

* Clean up codebase [zhongjiec] [[80b3e57](https://github.com/coryb/slipscheme/commit/80b3e57)]

## 0.0.3 - 2018-11-08

* Fix title logic and remove root reference [zhongjiec] [[3c12ffa](https://github.com/coryb/slipscheme/commit/3c12ffa)]
* Basic reference logic added [zhongjiec] [[b33608e](https://github.com/coryb/slipscheme/commit/b33608e)]
* add comment for golint [Cory Bennett] [[fe9b04e](https://github.com/coryb/slipscheme/commit/fe9b04e)]
* tweak struct naming for arrays [Cory Bennett] [[045a78f](https://github.com/coryb/slipscheme/commit/045a78f)]
* if file argument is "-" then read from stdin [Cory Bennett] [[b582d9a](https://github.com/coryb/slipscheme/commit/b582d9a)]
* print usage on no arguments [Cory Bennett] [[20ca3a3](https://github.com/coryb/slipscheme/commit/20ca3a3)]
* add flag to enable/disable comment generation [Cory Bennett] [[f6fa191](https://github.com/coryb/slipscheme/commit/f6fa191)]
* print comments for each type we write so golint will pass on the generated files [Cory Bennett] [[d9e2ce9](https://github.com/coryb/slipscheme/commit/d9e2ce9)]
* fix golint errors [Cory Bennett] [[5c13244](https://github.com/coryb/slipscheme/commit/5c13244)]

## 0.0.2 - 2016-08-07

* tweak header on printed documents [Cory Bennett] [[a09a964](https://github.com/coryb/slipscheme/commit/a09a964)]
* use address pointers when generating structs [Cory Bennett] [[461a302](https://github.com/coryb/slipscheme/commit/461a302)]
* cache processed types so we dont process them repeatedly [Cory Bennett] [[95f9a6e](https://github.com/coryb/slipscheme/commit/95f9a6e)]
* fix camel casing logic [Cory Bennett] [[f6d254f](https://github.com/coryb/slipscheme/commit/f6d254f)]
* add basic handling for '$refs' properties to reference common definitions [Cory Bennett] [[4268067](https://github.com/coryb/slipscheme/commit/4268067)]
* fix missing type for patternProperties [Cory Bennett] [[1b19afb](https://github.com/coryb/slipscheme/commit/1b19afb)]
* add -stdout option to print code to stdout rather than file [Cory Bennett] [[049f62f](https://github.com/coryb/slipscheme/commit/049f62f)]
* sort properties to esure consistent generation [Cory Bennett] [[8885172](https://github.com/coryb/slipscheme/commit/8885172)]

## 0.0.1 - 2016-08-06

* Initial Release
