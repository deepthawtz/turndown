turndown
========

`turndown` is useful when you want to scale down an Autoscaling Group by
a specific host identified by private IP address. Use-case could include
scaling down a Mesos cluster by specific hosts which have had all their tasks
drained/migrated and have been scheduled for maintenance.

## Install

`go install .`


```
$ turndown -h
Usage of turndown:
  -autoscaling-group-name string
        autoscaling group to target for scale down
  -down string
        specific IP to scale down within autoscaling group
  -region string
        AWS region (default "us-west-1")
```
