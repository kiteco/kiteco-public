import sys
import os
import string


def main():
	lookformore()
	
def lookfortheextrodinary():
	f = open("002-source.html")
	charcount = {}
	for shittystring in f:
		for achar in shittystring:
			if achar not in charcount:
				count = 1
				charcount[achar]=count
			else:
				count = charcount.pop(achar) + 1
				charcount[achar]=count
	thekeys = charcount.keys()
	for keys in thekeys:
		if charcount[keys] == 1:
			print keys
	f.close()
    """ dammit I hate you... you return the keys in alphabetical order"""

def lookformore():
	f = open("002-source.html")
	r = ''
	for thestring in f:
		for x in thestring:
			if x.isalpha():
				r += x
	print r
	f.close
			


if __name__ == '__main__':
	main()

