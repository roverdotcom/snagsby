import os
import unittest


class SnagsbyAcceptance(unittest.TestCase):
    def test_tricky_characters(self):
        # Note the trailing single quote
        expected = r'@^*309_!~``:*/\{}%()>$t' + "'"
        self.assertEqual(
            os.environ['TRICKY_CHARACTERS'],
            expected,
        )
        self.assertEqual(
            os.environ['RECURSIVE_TRICKY_CHARACTERS'],
            expected,
        )

    def test_starts_with_hash(self):
        self.assertEqual(
            os.environ['STARTS_WITH_HASH'],
            '#hello?world',
        )


if __name__ == '__main__':
    unittest.main()
