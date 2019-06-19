from argparse import ArgumentParser
from collections import defaultdict
import logging
import os
from glob import glob
import yaml
import warnings


import pandas as pd
from joblib import delayed, Parallel
from lookout.core.cmdline import ArgumentDefaultsHelpFormatterNoNone
from lookout.style.typos.analyzer import IdTyposAnalyzer
import textdistance
import spacy


def log(*args, logtype="debug", sep=" "):
    getattr(logging, logtype)(sep.join(str(a) for a in args))


def pipeline(yaml_dir, n_jobs=10):
    distance = textdistance.DamerauLevenshtein()

    yaml_files = glob(os.path.join(yaml_dir, "*"))
    log("Number of YAML files", len(yaml_files))

    HERC_COLUMNS = ["repository", "hash"]
    TYPOS_COLUMNS = ["wrong", "correct", "commit", "file", "line"]

    def yaml_to_dict(yaml_loc):
        if not yaml_loc.endswith("yaml"):
            # commits.txt
            return []
        rows = []
        with open(yaml_loc, "r") as f:
            a = yaml.load(f.read(), Loader=yaml.FullLoader)

        base = {col: a["hercules"][col] for col in HERC_COLUMNS}
        for typo in a["TyposDataset"]:
            res = base.copy()
            for col in TYPOS_COLUMNS:
                res[col] = typo[col]
            rows.append(res)
        return rows

    results = Parallel(n_jobs=n_jobs)(delayed(yaml_to_dict)(loc) for loc in yaml_files)
    pandas_dict = defaultdict(list)
    for rows in results:
        for row in rows:
            for c in (HERC_COLUMNS + TYPOS_COLUMNS):
                pandas_dict[c].append(row[c])
    df = pd.DataFrame.from_dict(pandas_dict)
    initial_n_samples = df.shape[0]
    log("Number of samples in initial dataset", initial_n_samples)
    # deduplication
    deduplicated_df = df.drop_duplicates(subset=["wrong", "correct"], keep="first")
    log("Number of samples after deduplication", deduplicated_df.shape[0], ", before",
        initial_n_samples)

    # check that number of subtokens keeps the same
    splitter = IdTyposAnalyzer.create_token_parser()

    def check_2(line):
        wrong = line.wrong
        correct = line.correct
        wrong_tokens = list(splitter.split(wrong))
        corr_tokens = list(splitter.split(correct))
        if len(wrong_tokens) != len(corr_tokens):
            return "Number of subtokens is different"
        if not len(wrong_tokens):
            return "Identifier without alphabetic characters"
        return ""

    deduplicated_df["check2"] = deduplicated_df.apply(check_2, axis=1)

    log("Number of good samples after check2",
        deduplicated_df[deduplicated_df["check2"] == ""].shape[0],
        ", before", initial_n_samples)

    # Demerau-Levenshtein distance
    def check_3(line):
        wrong = line.wrong
        correct = line.correct
        wrong_tokens = list(splitter.split(wrong))
        corr_tokens = list(splitter.split(correct))
        res = []
        for t, ct in zip(wrong_tokens, corr_tokens):
            if distance(t, ct) > 2:
                res.append((t, ct))
        if res:
            return "big Demerau-Levenshtein distance %s" % res
        return ""

    deduplicated_df["check3"] = deduplicated_df.apply(check_3, axis=1)
    suspicious_tokens = deduplicated_df[deduplicated_df["check3"] != ""]
    log("Number of samples with big Demerau-Levenshtein distance",
              suspicious_tokens.shape[0])

    # examples, where token splits of the wrong and the correct identifiers are equal
    # (they differ in non-alpha chars or casing)
    deduplicated_df["wrong_split"] = deduplicated_df["wrong"].apply(
        lambda x: " ".join(splitter.split(x)))
    deduplicated_df["correct_split"] = deduplicated_df["correct"].apply(
        lambda x: " ".join(splitter.split(x)))
    deduplicated_df["check4"] = ""
    deduplicated_df["check4"][
        deduplicated_df["wrong_split"] == deduplicated_df["correct_split"]] = "Bad split"
    log("Number of samples where tokens are the same",
        deduplicated_df[deduplicated_df["check4"] == "Bad split"].shape[0])
    # examples, where wrong and correct identifiers are equal on lemmas level.
    nlp = spacy.load("en", disable=["parser", "ner"])

    # Filter examples with equal lemmas
    def _lemmatize(token):
        lemm = nlp(token)
        if len(lemm) > 1 or lemm[0].lemma_ == "-PRON-" or (
                token[-2:] == "ss" and lemm[0].lemma_ == token[:-1]):
            return token
        return lemm[0].lemma_

    deduplicated_df["wrong_lem"] = deduplicated_df["wrong_split"].apply(
        lambda x: " ".join(_lemmatize(token) for token in x.split()))
    deduplicated_df["correct_lem"] = deduplicated_df["correct_split"].apply(
        lambda x: " ".join(_lemmatize(token) for token in x.split()))

    deduplicated_df["check5"] = ""
    deduplicated_df["check5"][(deduplicated_df["wrong_lem"] == deduplicated_df["correct_lem"])] = \
        "Equal lemmas"
    log("Number of good samples after check5",
              deduplicated_df[deduplicated_df["check5"] == ""].shape[0],
              ", before", initial_n_samples)

    deduplicated_df["check6"] = ""
    deduplicated_df["check6"][(deduplicated_df["wrong"].str.lower() ==
                               deduplicated_df["correct"].str.lower())] = \
        "Difference in case"

    good_df = deduplicated_df[
        (deduplicated_df["check2"] == "") & (deduplicated_df["check3"] == "") &
        (deduplicated_df["check4"] == "") & (deduplicated_df["check5"] == "") &
        (deduplicated_df["check6"] == "")
    ]
    good_df["repository"] = good_df["repository"].str.replace("@", "/")
    log("Number of good samples", good_df.shape[0])
    for i, row in good_df[["repository"] + TYPOS_COLUMNS].iterrows():
        print(",".join(map(str, row.values)))


if __name__ == "__main__":
    parser = ArgumentParser(formatter_class=ArgumentDefaultsHelpFormatterNoNone)
    parser.add_argument("-i", "--yaml-dir", help="Directory with YAML files.")
    parser.add_argument("-n", "--ncores", type=int, default=10, help="Number of cores to use.")
    parser.add_argument("--log-level", default="INFO", choices=logging._nameToLevel,
                        help="Logging verbosity.")
    args = parser.parse_args()
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        pipeline(yaml_dir=args.yaml_dir, n_jobs=args.ncores)
