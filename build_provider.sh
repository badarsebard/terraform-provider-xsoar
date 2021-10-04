go fmt
go build -o terraform-provider-xsoar
arch=""
case $(uname -m) in
    i386)   arch="386" ;;
    x86_64) arch="amd64" ;;
esac
os=""
case $(uname) in
    [Dd]arwin)   os="darwin" ;;
    [Ww]indows)  os="windows" ;;
    [Ll]inux)    os="linux" ;;
esac
mv terraform-provider-xsoar ~/.terraform.d/plugins/local/badarsebard/xsoar/0.1.0/${os}_${arch}/terraform-provider-xsoar
rm -rf examples/.terraform/ examples/.terraform.lock.hcl examples/*.tfstate
