# Third-party software notices

This project is distributed under the GNU General Public License
version 3 (GPL-3.0), see [LICENSE](LICENSE).

It includes or redistributes third-party software components
distributed under their own licenses.

## DDS — Double Dummy Solver

This project uses the Double Dummy Solver (DDS) developed for the
Bridge card game. It powers the "Calcul du PAR" feature of the web
client, which reports the double dummy trick count of each side in
each strain.

Project:
https://github.com/dds-bridge/dds

License:
Apache License, Version 2.0 (Apache-2.0)

### What is redistributed here

This repository does **not** contain the DDS source code. It
redistributes DDS compiled to WebAssembly, as two generated files:

| File | What it is |
|---|---|
| [cli/dds_web_wasm_bin.js](cli/dds_web_wasm_bin.js) | the compiled DDS WebAssembly module, base64-encoded |
| [cli/dds_web_wasm.js](cli/dds_web_wasm.js) | the Emscripten glue that instantiates it |

Both are build artifacts derived from the DDS source, and remain
subject to the terms of the Apache License, Version 2.0.

The Apache License, Version 2.0 is available at:

https://www.apache.org/licenses/LICENSE-2.0

DDS includes historical copyright notices including:

Copyright 2006-2014 by Bo Haglund
Copyright 2014 by Bo Haglund / Soren Hein as of version 2.8.0.

## Relationship with this project

The `encheres-bridge` project itself is licensed under the GNU General
Public License, version 3.

The Apache-2.0 licensed DDS components remain subject to the terms of
the Apache License, Version 2.0. Their inclusion in this GPL-3.0
project does not change the licensing terms applicable to the DDS
components.
