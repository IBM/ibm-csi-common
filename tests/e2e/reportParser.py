import json
import argparse
import subprocess
import os

repoHome=os.environ['GOPATH']+ "/src/github.com/IBM/ibm-csi-common"
finalResultFile=repoHome + "/block-vpc-csi-driver-e2e.out"
confFile=repoHome +"/tests/e2e/conf/testcases.json"

resultFile=open(finalResultFile, "w+")
with open(confFile) as f:
   data = json.load(f)
f.close()

def executetask(cmd):
        # Internal method to execute a command
        # RC 0 - Cmd execution with zero return code
        # RC 1 - Cmd executed with non zero return code
        # RC 2 - Runtime error / exception
        cmd_output = ''
        cmd_err = ''
        rc = 0

        #print ("CommmandExec \"{0}\"".format(cmd))
        try:
            process = subprocess.Popen(cmd, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, shell=True)
            process.wait()
            (cmd_output, cmd_err) = process.communicate()
        except Exception as err:
            print ("Command \"{0}\" execution failed. ERROR: {1}".format(cmd, str(err)))
            cmd_err = "Command execution failed"
            rc = 2
        else:
            if process.returncode == 0:
                rc = 0
            else:
                rc = 1
            if cmd_err:
               print ("{0}\nERROR: {1}".format(cmd, cmd_err.strip()))
        return (rc, cmd_output, cmd_err)

def readAndWriteTestCases():
    for sets in data['setup']:
        resultFile.write("%s: NA\n" %(sets))
        print sets
    for e2e in data['e2e']:
        resultFile.write("%s: NA\n" %(e2e))
        print e2e

def updateTestExecutionStatus(filename):
    with open(filename, "r") as fp:
        testList = fp.read().replace('\n', '')
    fp.close()
    for e2e in data['setup']:
        if (testList.find(e2e) == -1):
           print "Not able to find " + e2e
           command = "sed -i \"s/{testcase}: .*$/{testcase}: FAIL/g\" {path}".format(testcase=e2e, path=finalResultFile)
           (rc, cmd_out, cmd_err) = executetask(command)
           print command
	   print "Setup part failed. Test cases may not have executed"
           os.exit(1)
        else:
           print "Found " + e2e
           command = "sed -i \"s/{testcase}: .*$/{testcase}: PASS/g\" {path}".format(testcase=e2e, path=finalResultFile)
           print command
           (rc, cmd_out, cmd_err) = executetask(command)
    for e2e in data['e2e']:
        if (testList.find(e2e) == -1):
            print "Not able to find " + e2e
            command = "sed -i \"s/{testcase}: .*$/{testcase}: FAIL/g\" {path}".format(testcase=e2e, path=finalResultFile)
            (rc, cmd_out, cmd_err) = executetask(command)
            print command
        else:
            print "Found " + e2e
            command = "sed -i \"s/{testcase}: .*$/{testcase}: PASS/g\" {path}".format(testcase=e2e, path=finalResultFile)
            print command
            (rc, cmd_out, cmd_err) = executetask(command)


if __name__== "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("testListPath", help="path of the test list file created during e2e execution", type=str)
    args = parser.parse_args()
    path = args.testListPath

    readAndWriteTestCases()
    resultFile.close()
    updateTestExecutionStatus(path)
