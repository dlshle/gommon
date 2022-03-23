#!/bin/sh

cp $(go env GOROOT)/misc/wasm/wasm_exec.js ./
GOOS=js GOARCH=wasm go build -o main.wasm

touch index.html

cat>index.html<<EOF
<!doctype html>
<!--
Copyright 2018 The Go Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.
-->
<html>

<head>
        <meta charset="utf-8">
        <title>Go wasm</title>
</head>

<body>
        <!--
        Add the following polyfill for Microsoft Edge 17/18 support:
        <script src="https://cdn.jsdelivr.net/npm/text-encoding@0.7.0/lib/encoding.min.js"></script>
        (see https://caniuse.com/#feat=textencoder)
        -->
        <script src="wasm_exec.js"></script>
  <script>
    const go = new Go();
    console.log(go);
    WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
      .then((result) => go.run(result.instance));
  </script>

</body>
</html>
EOF
