import os
import subprocess
import json
import unittest


class SnagsbyAcceptanceTestCase(unittest.TestCase):
    def run_snagsby(self, source):
        return subprocess.check_output(
            [
                os.environ['SNAGSBY_BIN'],
                '-o=json',

            ] + source,
        )

    def get_json(self, source):
        return json.loads(self.run_snagsby(source))


class SnagsbyAcceptance(SnagsbyAcceptanceTestCase):
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

    def test_json_parsing(self):
        out = self.get_json([os.environ['SNAGSBY_E2E_SOURCE']])
        self.assertEqual(out['STARTS_WITH_HASH'], '#hello?world')

    def test_splat_source(self):
        out = self.get_json([
            'sm://snagsby/splat-tests/*'
        ])
        self.assertEqual(out['ONE'], 'one')
        self.assertEqual(out['TWO'], 'two')


if __name__ == '__main__':
    unittest.main()
