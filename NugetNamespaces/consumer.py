import argparse
from collections import defaultdict
import json
import logging
import os
from pathlib import Path
import shutil
import subprocess
from xml.dom import minidom

from pystalk import BeanstalkClient


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-b", "--beanstalkd", default="0.0.0.0:11300",
                        help="beanstalkd host:port.")
    parser.add_argument("-x", "--extractor", required=True,
                        help="Namespaces extractor executable path.")
    parser.add_argument("-t", "--tmp", required=True,
                        help="Temporary files directory.")
    parser.add_argument("-o", "--output", required=True,
                        help="Output file path.")
    return parser.parse_args()


def run_cmd(log, *cmd):
    p = subprocess.run(list(cmd), stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    output = p.stdout.decode()
    try:
        p.check_returncode()
    except Exception as e:
        log.info("\"%s\"", "\" \"".join(cmd))
        log.info("\n%s", output)
        raise e from None
    return output


def execute_job(job, exe, tmp, output, log):
    subdirs = run_cmd(log, "ls", "-1vr", job).split("\n")
    if not subdirs:
        log.warning("- (subdir) %s", job)
        return
    subdir = Path(job) / subdirs[0]
    nupkg = list(subdir.rglob("*.nupkg"))
    if not nupkg:
        log.warning("- (nupkg) %s", job)
        return
    nupkg = str(nupkg[0])
    nuspec = list(subdir.glob("*.nuspec"))
    if not nuspec:
        log.warning("- (nuspec) %s", job)
        return
    nuspec = str(nuspec[0])
    nuspec = minidom.parse(nuspec)
    name = nuspec.getElementsByTagName("id")[0].firstChild.nodeValue
    try:
        tags = nuspec.getElementsByTagName("tags")[0].firstChild.nodeValue
    except (IndexError, AttributeError):
        tags = ""
    try:
        descr = nuspec.getElementsByTagName("description")[0].firstChild.nodeValue
    except (IndexError, AttributeError):
        descr = ""
    tmp = Path(tmp) / str(os.getpid())
    tmp = tmp / name
    tmp.mkdir(parents=True, exist_ok=True)
    try:
        run_cmd(log, "unzip", "-o", "-d", str(tmp), nupkg)
        namespaces = defaultdict(int)
        for dll in tmp.rglob("*.dll"):
            try:
                for line in run_cmd(log, exe, str(dll)).split("\n"):
                    if not line:
                        continue
                    ns, count = line.split()
                    namespaces[ns] += int(count)
            except subprocess.CalledProcessError:
                log.warning("failed to extract %s", dll)
        json.dump({"name": name, "tags": tags, "description": descr, "namespaces": namespaces},
                  output, sort_keys=True)
        output.write("\n")
    finally:
        shutil.rmtree(tmp)
    log.info("✔ %s", job)


def main():
    args = parse_args()
    log = logging.getLogger("nuget-meta")
    logging.basicConfig(level=logging.INFO)
    host, port = args.beanstalkd.split(":")
    client = BeanstalkClient(host, int(port), auto_decode=True)
    try:
        with open(args.output, "a") as fout:
            for job in client.reserve_iter():
                try:
                    execute_job(job.job_data, args.extractor, args.tmp, fout, log)
                except Exception:
                    log.exception(job)
                    try:
                        client.bury_job(job.job_id)
                    except Exception as e:
                        log.error("bury %s: %s: %s", job.job_data, type(e).__name__, e)
                    continue
                try:
                    client.delete_job(job.job_id)
                except Exception as e:
                    log.error("delete %s: %s: %s", job.job_data, type(e).__name__, e)
    finally:
        shutil.rmtree(os.path.join(args.tmp, str(os.getpid())), ignore_errors=True)


if __name__ == "__main__":
    exit(main())
