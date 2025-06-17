head -c 1048576 </dev/urandom > random-file.txt

kubectl create configmap example-configmap --from-file=random-file.txt


for i in {1..10}; do
  kubectl create configmap example-configmap-$i --from-file=random-file.txt
done

for i in {1..10}; do
  kubectl create secret generic example-secret-$i --from-file=random-file.txt
done
