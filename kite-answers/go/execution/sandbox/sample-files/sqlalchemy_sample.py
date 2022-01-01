# a file containing a sqlalchemy mapped class, User
import sqlalchemy
from sqlalchemy import Column, Integer, String, ForeignKey
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker

engine = sqlalchemy.create_engine("sqlite:///:memory:")
session = sessionmaker(bind=engine)()
base = declarative_base()

class User(base):
    __tablename__ = "users"
    uid = Column(Integer, primary_key=True)
    username = Column(String)
    def __repr__(self):
        return "User:" + self.username

class Car(base):
    __tablename__ = "cars"
    cid = Column(Integer, primary_key=True)
    model = Column(String)
    def __repr__(self):
        return "Car:" + self.model 
        
    owner_name = Column(Integer, ForeignKey("users.username"))

base.metadata.create_all(engine) 

session.add_all([ 
    User(username="Andrew"),
    User(username="Barbara"),
    User(username="Casey"),
    Car(owner_name="Andrew", model="Accord"),
    Car(owner_name="Casey", model="Corvette" )
    ])