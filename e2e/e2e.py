import os
import subprocess
import unittest
import json


class SnagsbyTestCase(unittest.TestCase):
    def run_snagsby(self, sources, output=None):
        if not output:
            output = 'json'

        return subprocess.check_output([
            os.environ['SNAGSBY_BIN'],
            '-o',
            output,
        ] + sources)

    def run_snagsby_json(self, sources):
        out = self.run_snagsby(sources, 'json')
        return json.loads(out)


class SnagsbyAcceptance(SnagsbyTestCase):
    def test_tricky_characters(self):
        # Note the trailing single quote
        expected = r'@^*309_!~``:*/\{}%()>$t' + "'"
        self.assertEqual(
            os.environ['TRICKY_CHARACTERS'],
            expected,
        )

    def test_starts_with_hash(self):
        self.assertEqual(
            os.environ['STARTS_WITH_HASH'],
            '#hello?world',
        )

    def test_nested_key_name(self):
        """
        Useful for testing sources that search paths like ssm
        """
        self.assertEqual(
            os.environ['NESTED_PARAM'],
            'nested',
        )


if __name__ == '__main__':
    unittest.main()
