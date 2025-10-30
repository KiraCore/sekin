#!/usr/bin/env bash
set -e
set -x

go get github.com/cosmos/cosmos-sdk@v0.47.6

rm -rfv ./proto-gen
mkdir -p ./proto-gen
mkdir -p ./proto-gen ./proto
cosmos_sdk_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-sdk@v0.47.6)

rm -rfv ./third_party/proto ./proto/cosmos ./proto/kira
mkdir -p ./third_party/proto
cp -rfv $cosmos_sdk_dir/proto/cosmos ./proto
cp -rfv $cosmos_sdk_dir/proto/amino ./proto
tar -C ./third_party/ -xvf ./third_party_cosmos_proto.tar.xz

wget https://github.com/KiraCore/sekai/releases/download/v0.4.5/source-code.tar.gz
mkdir -p sekai
tar -C ./sekai/ -xvf ./source-code.tar.gz
rm -rfv ./source-code.tar.gz
cp -rfv ./sekai/proto/kira ./proto
rm -rfv ./sekai
rm -rfv ./codec && mkdir -p codec/types
buf protoc -I "third_party/proto" --gogotypes_out=./codec/types third_party/proto/google/protobuf/any.proto
mv codec/types/google.golang.org/protobuf/types/known/anypb/any.pb.go codec/types
rm -rfv codec/types/google.golang.org/protobuf/types/known/anypb
rm -rfv ./third_party/proto/gogoproto
rm -rfv ./third_party/proto/google

sed '/proto\.RegisterType/d' codec/types/any.pb.go > tmp && mv tmp codec/types/any.pb.go

proto_dirs=$(find ./proto -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)

fil=./proto/cosmos/base/v1beta1/coin.proto && \
 sed -i="" 's/ = \"github.com\/cosmos\/cosmos-sdk\/types/ = \"github.com\/saiset-co\/sai-interx-manager\/proto-gen\/cosmos\/base\/v1beta1/g' "$fil" || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
for dir in $proto_dirs; do
    proto_fils=$(find "${dir}" -maxdepth 1 -name '*.proto')
    for fil in $proto_fils; do
        sed -i="" 's/, (gogoproto.castrepeated) = \"github.com\/cosmos\/cosmos-sdk\/types.Coins\"//g' "$fil" || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
        sed -i="" 's/github.com\/cosmos\/cosmos-sdk\/x/github.com\/saiset-co\/sai-interx-manager\/proto-gen\/cosmos/g' "$fil" || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
        sed -i="" 's/\[(gogoproto.stdtime) = true, (gogoproto.nullable) = false\]/\[(gogoproto.stdtime) = true, (gogoproto.nullable) = false, (gogoproto.moretags) = \"yaml:\\\"date\\\"\"\]/g' "$fil" || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
        sed -i="" 's/github.com\/saiset-co\/sai-interx-manager\/proto-gen\/cosmos\/auth\/types/github.com\/saiset-co\/sai-interx-manager\/proto-gen\/cosmos\/auth\/v1beta1/g' "$fil" || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
    done
done

sed -i="" 's/message IdentityRecord {/message IdentityRecord \{\n  option (gogoproto.goproto_getters) = false;/g' ./proto/kira/gov/identity_registrar.proto || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
sed -i="" 's/ \[(cosmos_proto.accepts_interface) = \"AccountI\"\]//g' ./proto/cosmos/auth/v1beta1/query.proto || ( echoErr "ERROR: Failed to sed file: '$fil'" && exit 1 )
cp -rfv ./proto-override/cosmos/auth/v1beta1/auth.proto ./proto/cosmos/auth/v1beta1
cp -rfv ./proto-override/kira/tokens/query.proto ./proto/kira/tokens/query.proto

for dir in $proto_dirs; do
    proto_fils=$(find "${dir}" -maxdepth 1 -name '*.proto') 
    for fil in $proto_fils; do
        if grep -q "option go_package" "$fil"; then
            buf protoc \
            -I "./proto" \
            -I third_party/grpc-gateway/ \
            -I third_party/googleapis/ \
            -I third_party/proto/ \
            --go_out=paths=source_relative:./proto-gen \
            --go-grpc_out=paths=source_relative:./proto-gen \
            --grpc-gateway_out=logtostderr=true,paths=source_relative:./proto-gen \
            $fil || ( echoErr "ERROR: Failed proto build for: ${fil}" && sleep 2 && exit 1 )
        fi
    done
done
