from typing import List, Type, Union

import sqlalchemy
from sqlalchemy import Column, Float, ForeignKey, Integer, LargeBinary, Text, UniqueConstraint
from sqlalchemy.orm import relationship
from sqlalchemy.ext.declarative import declarative_base


Base = declarative_base()


class Experiment(Base):
    __tablename__ = "experiments"

    id = Column(Integer, primary_key=True)
    name = Column(Text, unique=True)
    description = Column(Text)


class User(Base):
    __tablename__ = "users"

    id = Column(Integer, primary_key=True)
    login = Column(Text, unique=True)
    username = Column(Text)
    avatar_url = Column(Text)
    role = Column(Text)


class Pair(Base):
    __tablename__ = "pairs"

    id = Column(Integer, primary_key=True)
    blob_id_a = Column(Text)
    repository_id_a = Column(Text)
    commit_hash_a = Column(Text)
    path_a = Column(Text)
    content_a = Column(Text)
    hash_a = Column(Text)
    blob_id_b = Column(Text)
    repository_id_b = Column(Text)
    commit_hash_b = Column(Text)
    path_b = Column(Text)
    content_b = Column(Text)
    hash_b = Column(Text)
    score = Column(Float)
    experiment_id = Column(ForeignKey("experiments.id"))
    uast_a = Column(LargeBinary)
    uast_b = Column(LargeBinary)

    experiment = relationship("Experiment")


class Assignment(Base):
    __tablename__ = "assignments"
    __table_args__ = (
        UniqueConstraint("user_id", "pair_id", "experiment_id"),
    )

    id = Column(Integer, primary_key=True)
    user_id = Column(ForeignKey("users.id"))
    pair_id = Column(ForeignKey("pairs.id"))
    experiment_id = Column(ForeignKey("experiments.id"))
    answer = Column(Text)
    duration = Column(Integer)

    experiment = relationship("Experiment")
    pair = relationship("Pair")
    user = relationship("User")


def _smart_str(data) -> Union[str, bytes]:
    try:
        return data.decode("utf-8")
    except UnicodeDecodeError:
        return bytes(data)


class DuplicatesDataset:
    def __init__(self, db_file_name: str):
        self._engine = sqlalchemy.create_engine("sqlite:///" + db_file_name)
        self._cache = {}

    @property
    def experiments(self) -> List[Experiment]:
        return self._select_all(Experiment)

    @property
    def users(self) -> List[User]:
        return self._select_all(User)

    @property
    def pairs(self) -> List[Pair]:
        return self._select_all(Pair)

    @property
    def assignments(self) -> List[Assignment]:
        return self._select_all(Assignment)

    def __enter__(self) -> sqlalchemy.engine.base.Connection:
        """
        Return a connection to the database which can execute queries.
        """
        c = self._engine.connect()
        c.connection.connection.text_factory = _smart_str
        return c

    def __exit__(self, exc_type, exc_val, exc_tb):
        pass

    def _select_all(self, cls: Type[Base]) -> List[Base]:
        cached = self._cache.get(cls)
        if cached is None:
            with self as c:
                cached = list(c.execute(sqlalchemy.sql.select([cls])))
            self._cache[cls] = cached
        return cached
