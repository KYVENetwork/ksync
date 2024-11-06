if ./build/ksync block-sync -b $HOME/bins/kyved-v1.0.0 -c kaon-1 -t 50 -r -y; then
  echo "success"
else
  echo "error"
fi