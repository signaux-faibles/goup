#!/usr/bin/python2
# -*- coding: utf-8 -*-

import json, csv
import sys, os

def main():
  if len(sys.argv) != 2:
    print "goup.py permet de lister le contenu d'un répertoire de stockage goup"
    print "usage: goupy.py [path of directory]"
    sys.exit(1)

  if not (os.path.isdir(sys.argv[1])):
    print "Le chemin indiqué n'est pas un répertoire"
    sys.exit(1)

  data = [read_info(sys.argv[1] + path) for path in os.listdir(sys.argv[1]) if  read_info(sys.argv[1] + path) != None]
  
  writer = csv.DictWriter(sys.stdout, get_keys(data), dialect=csv.excel)
  writer.writeheader() 
  for d in data:
    writer.writerow(d)

def read_info(path):
  with open(path) as json_file:
    data = json.load(json_file)
    for k in data['MetaData'].keys():
      data['md-' + k] = data['MetaData'][k]
      del data['MetaData'][k]
    del data['MetaData']

    return data

def get_keys(datas):
  keys = set()

  for d in datas:
    keys = keys.union(d.keys())

  return keys

if __name__=="__main__":
  main()