curl --digest -u admin:secret \
   -X POST \
   -H "Content-Type: text/plain" \
   --data-binary @retail-schema.proto \
   http://localhost:11222/rest/v2/caches/___protobuf_metadata/retail-schema.proto
