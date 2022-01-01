import argparse

from model.feeder import FileDataFeeder, Feed


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--samples_dir', type=str, default='out/train_samples')
    parser.add_argument('--batch_size', type=int, default=20)

    args = parser.parse_args()

    feeder = FileDataFeeder(args.samples_dir, batch_size=args.batch_size)

    while True:
        _ = feeder.next()


if __name__ == '__main__':
    main()
