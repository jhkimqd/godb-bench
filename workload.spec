# Initial dataset size
recordcount=10000
# Number of operations to do after setting up initial dataset
operationcount=10000
workload=core
warmuptime=1

readallfields=true

readproportion=0.25
updateproportion=0.25
scanproportion=0
insertproportion=0.5

requestdistribution=uniform

# Enable batch operations (default is 1, meaning no batching)
# batch.size=100