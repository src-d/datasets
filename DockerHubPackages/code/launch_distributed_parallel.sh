for process in $(seq 0 $(($1 - 1)))
do
	ipython main.py ./images.txt ./packages $process $1 &
done
