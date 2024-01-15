set -xe

BASEDIR=$(dirname $0)
ROOTDIR=$(builtin cd $BASEDIR/..; pwd)
MGCDIR=${MGCDIR:-"mgc/cli"}
OAPIDIR=${OAPIDIR:-"$MGCDIR/openapis"}

python3 $ROOTDIR/scripts/oapi_check_links.py $OAPIDIR
