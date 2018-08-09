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


if __name__ == '__main__':
    unittest.main()
