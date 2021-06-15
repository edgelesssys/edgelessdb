#!/usr/bin/env python3

import json

manifestIn = 'manifest-template.json'
manifestOut = 'manifest.json'
caFile = 'owner/ca-cert.pem'

with open(caFile, 'r') as f:
    ca = f.read()
with open(manifestIn, 'r') as f:
    j = json.load(f)
j["ca"] = ca
with open(manifestOut, 'w') as f:
    json.dump(j, f, indent=4)
