#!/bin/sh
# TODO allow to test with more than one hadoop version if needed
HADOOP_VERSION=${HADOOP_VERSION-"2.7.4"}

HADOOP_HOME="/tmp/hadoop-$HADOOP_VERSION"
NN_PORT="9000"
HADOOP_NAMENODE="localhost:$NN_PORT"
HADOOP_URL="http://apache.mirrors.tds.net/hadoop/common/hadoop-$HADOOP_VERSION/hadoop-$HADOOP_VERSION.tar.gz"

if [ ! -d "$HADOOP_HOME" ]; then
  mkdir -p $HADOOP_HOME

  echo "Downloading Hadoop from $HADOOP_URL to ${HADOOP_HOME}/hadoop.tar.gz"
  curl -o ${HADOOP_HOME}/hadoop.tar.gz -L $HADOOP_URL

  echo "Extracting ${HADOOP_HOME}/hadoop.tar.gz into $HADOOP_HOME"
  tar zxf ${HADOOP_HOME}/hadoop.tar.gz --strip-components 1 -C $HADOOP_HOME
fi

MINICLUSTER_JAR=$(find $HADOOP_HOME -name "hadoop-mapreduce-client-jobclient*.jar" | grep -v tests | grep -v sources | head -1)
if [ ! -f "$MINICLUSTER_JAR" ]; then
  echo "Couldn't find minicluster jar"
  exit 1
fi
echo "minicluster jar found at $MINICLUSTER_JAR"


# start the namenode in the background
echo "Starting hadoop namenode..."
$HADOOP_HOME/bin/hadoop jar $MINICLUSTER_JAR minicluster -nnport $NN_PORT -datanodes 3 -nomr -format "$@" > minicluster.log 2>&1 &
sleep 30

HADOOP_FS="$HADOOP_HOME/bin/hadoop fs -Ddfs.block.size=1048576"

export HADOOP_NAMENODE='$HADOOP_NAMENODE'