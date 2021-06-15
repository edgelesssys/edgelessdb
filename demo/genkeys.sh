set -e
mkdir owner reader writer
openssl req -x509 -newkey rsa -nodes -subj '/CN=Owner CA' -keyout owner/ca-key.pem -out owner/ca-cert.pem
openssl req -newkey rsa -nodes -subj '/CN=Reader' -keyout reader/key.pem -out csr-reader.pem
openssl x509 -req -CA owner/ca-cert.pem -CAkey owner/ca-key.pem -CAcreateserial -in csr-reader.pem -out reader/cert.pem
openssl req -newkey rsa -nodes -subj '/CN=Writer' -keyout writer/key.pem -out csr-writer.pem
openssl x509 -req -CA owner/ca-cert.pem -CAkey owner/ca-key.pem -in csr-writer.pem -out writer/cert.pem
rm csr-reader.pem csr-writer.pem
./genkeys-addtomanifest.py
