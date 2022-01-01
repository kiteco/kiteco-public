"""Script to download reports from TestRail and convert them to PDFs to be pushed to Dropbox"""

import json
import zipfile
import shutil
import os
import time
import datetime
import requests
import pdfkit
import slacker

# JSON file with download info
TESTRAIL_JSON = "testrail.json"
REPORT_NAME = "report-{}"
DROPBOX_DIR = os.path.expanduser("~/dropbox_kite/Shared/Testrail Reports/")

# slack interface
slack = slacker.Slacker(os.environ["SLACK_TOKEN"])

def download_report(report_id=None):
    """Download a testrail report as zip and return file name"""

    tr_info = json.load(open(TESTRAIL_JSON))
    # make URL using ID; use latest from JSON if None
    if report_id is None:
        report_id = tr_info["latest_report_id"]
    url = tr_info["dl_url"].format(id=report_id)

    # download with cookie info from JSON
    r = requests.get(url, cookies={"tr_session": tr_info["tr_session"]})
    content = r.content

    # save content to zip file
    timestamp = date_from_report_id(report_id)
    filename = REPORT_NAME.format(timestamp) + ".zip"
    with open(filename, "bw+") as f:
        f.write(content)

    return f.name

def zip_to_pdf(filename):
    """Unzips report and converts into a PDF"""

    # check if invalid zip - happens when report does not exist
    if not zipfile.is_zipfile(filename):
        raise ValueError("Invalid zip file")

    # if valid, unzip all to folder with same name as zip
    dirname = filename[:-len(".zip")]
    with zipfile.ZipFile(filename) as f:
        f.extractall(dirname)

    # create pdf
    pdf_filename = dirname + ".pdf"
    pdfkit.from_file(os.path.join(dirname, "index.html"), pdf_filename)

    return pdf_filename

def to_dropbox(report):
    """Copy report to dropbox and replace the latest report with this one"""

    # copy new report
    shutil.copy2(report, DROPBOX_DIR)
    # replace latest
    shutil.copy2(report, os.path.join(DROPBOX_DIR, "latest.pdf"))

def to_slack(report):
    """Post report to slack"""
    # test channel
    # ch_id = "C1FM10CSK"
    # release channel
    ch_id = "C0M76GA13"

    # upload file and check response
    r = slack.files.upload(report, channels=[ch_id])
    if not r.successful:
        print(r.error)

def date_from_report_id(report_id):
    """Get the date of the report from the report ID

    Report IDs are sequential and one has been generated each day

    The start date is based off of the fact that the first scheduled report generated on 2017-05-25
    had report ID 17
    """
    start_date = datetime.datetime(2017, 5, 7)
    report_date = start_date + datetime.timedelta(days=report_id)
    date_format = "%Y-%m-%d"

    return report_date.strftime(date_format)

def update_tr_info():
    """After successful export, update testrail info JSON"""

    tr_info = json.load(open(TESTRAIL_JSON))
    # increment report id
    tr_info["latest_report_id"] += 1
    # write it back
    json.dump(tr_info, open(TESTRAIL_JSON, "w"))

def run():
    """Once a day, check for the latest report and write it if it exists"""
    while 1:
        # if outside of posting time, continue
        dt = datetime.datetime.now()
        if not (dt.weekday in (1, 2, 3) and dt.hour == 19):
            continue

        f = download_report()
        try:
            report = zip_to_pdf(f)
        # only do the rest if valid zip is downloaded
        except ValueError:
            print("Invalid zip")
        else:
            to_dropbox(report)
            to_slack(report)
            update_tr_info()
            print("Uploaded {} to Dropbox".format(report))

        time.sleep(60*60)


if __name__ == "__main__":
    run()
