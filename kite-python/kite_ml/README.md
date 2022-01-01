# Setup

- Ensure python 3 is installed
- Ensure pip is installed
- Create a virtual env

  OSX
   ```$xslt
  python3 -m venv env
  ```
  Azure (pip is not installed by default?)
   ```$xslt
  python3.6 -m venv env --without-pip
  ```
- Activate virtual env
```$xslt
source env/bin/activate
```
  Azure - install pip in the venv
  ```$xslt
  curl https://bootstrap.pypa.io/get-pip.py | python
  ```
- Install requirements
```$xslt
pip install -r requirements.txt
```
- Install Kite ML in venv. If you're developing and want changes to take immediate effect, replace "install" with "develop"
```$xslt
./setup.py install
```

# Testing
- Make sure you are in an activated Virtual env
- Run `pytest` from this directory