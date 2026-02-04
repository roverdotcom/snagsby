import os
import subprocess
import unittest
import json
import sys


class SnagsbyAcceptance(unittest.TestCase):
    """Tests for Snagsby using actual AWS Secrets Manager integration."""

    def test_tricky_characters(self):
        """Test that special characters are properly escaped in environment variables."""
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
        """Test that values starting with # are handled correctly."""
        self.assertEqual(
            os.environ['STARTS_WITH_HASH'],
            '#hello?world',
        )


class SnagsbyCliTests(unittest.TestCase):
    """Tests for Snagsby CLI functionality."""

    def setUp(self):
        """Set up test fixtures."""
        self.snagsby_path = self._get_snagsby_path()
        self.test_source = os.environ.get('SNAGSBY_E2E_SOURCE', 'sm://snagsby/acceptance')

    def _get_snagsby_path(self):
        """Get the path to the snagsby binary."""
        os_name = os.uname().sysname.lower()
        return f'./dist/{os_name}/snagsby'

    def test_version_flag(self):
        """Test that -v flag prints version information."""
        result = subprocess.run(
            [self.snagsby_path, '-v'],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        self.assertIn('snagsby version', result.stdout)
        self.assertIn('aws sdk', result.stdout)
        self.assertIn('golang', result.stdout)

    def test_json_output_format(self):
        """Test JSON output format (-o json)."""
        sources = self.test_source.split()
        result = subprocess.run(
            [self.snagsby_path, '-o', 'json'] + sources,
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        # Should be valid JSON
        try:
            data = json.loads(result.stdout)
            self.assertIsInstance(data, dict)
            # Check for expected keys from the test secrets
            self.assertIn('TRICKY_CHARACTERS', data)
            self.assertIn('STARTS_WITH_HASH', data)
        except json.JSONDecodeError:
            self.fail(f"Output is not valid JSON: {result.stdout}")

    def test_env_output_format(self):
        """Test default env output format."""
        sources = self.test_source.split()
        result = subprocess.run(
            [self.snagsby_path] + sources,
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        # Should contain export statements
        self.assertIn('export ', result.stdout)
        self.assertIn('TRICKY_CHARACTERS=', result.stdout)
        self.assertIn('STARTS_WITH_HASH=', result.stdout)

    def test_envfile_output_format(self):
        """Test envfile output format (no export prefix)."""
        sources = self.test_source.split()
        result = subprocess.run(
            [self.snagsby_path, '-o', 'envfile'] + sources,
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        # Should NOT contain export statements
        self.assertNotIn('export ', result.stdout)
        # Should contain variable assignments
        self.assertIn('TRICKY_CHARACTERS=', result.stdout)
        self.assertIn('STARTS_WITH_HASH=', result.stdout)

    def test_fail_on_error_flag(self):
        """Test -e flag (fail on errors)."""
        # Test with an invalid source
        result = subprocess.run(
            [self.snagsby_path, '-e', 'sm://nonexistent/secret/that/does/not/exist'],
            capture_output=True,
            text=True
        )
        # Should exit with non-zero code
        self.assertNotEqual(result.returncode, 0)
        # Should have error message in stderr
        self.assertIn('Error processing snagsby source', result.stderr)

    def test_invalid_formatter(self):
        """Test behavior with invalid output format."""
        sources = self.test_source.split()
        result = subprocess.run(
            [self.snagsby_path, '-o', 'invalid_format'] + sources,
            capture_output=True,
            text=True
        )
        # Should exit with code 2
        self.assertEqual(result.returncode, 2)
        self.assertIn('No formatter found', result.stderr)

    def test_show_summary_flag(self):
        """Test -show-summary flag."""
        sources = self.test_source.split()
        result = subprocess.run(
            [self.snagsby_path, '-show-summary'] + sources,
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        # Summary should be in stderr
        # Should show source URLs and item counts
        for source in sources:
            # The summary line format is: URL (count) => (keys)
            self.assertTrue(
                any(source in line for line in result.stderr.split('\n')),
                f"Expected source '{source}' in summary output"
            )

    def test_multiple_sources(self):
        """Test merging multiple sources with later sources overriding earlier ones."""
        sources = self.test_source.split()
        if len(sources) > 1:
            result = subprocess.run(
                [self.snagsby_path, '-o', 'json'] + sources,
                capture_output=True,
                text=True
            )
            self.assertEqual(result.returncode, 0)
            data = json.loads(result.stdout)
            # Should have keys from both sources
            self.assertIsInstance(data, dict)
            self.assertGreater(len(data), 0)

    def test_recursive_secrets_manager_pattern(self):
        """Test recursive Secrets Manager pattern (wildcard)."""
        # Test the recursive pattern if it's in the test sources
        recursive_source = 'sm:///snagsby/app/acceptance/*'
        if recursive_source in self.test_source:
            result = subprocess.run(
                [self.snagsby_path, recursive_source],
                capture_output=True,
                text=True
            )
            self.assertEqual(result.returncode, 0)
            # Should have RECURSIVE_ prefixed keys
            self.assertIn('RECURSIVE_TRICKY_CHARACTERS', result.stdout)

    def test_single_secrets_manager_pattern(self):
        """Test single Secrets Manager secret (non-wildcard)."""
        single_source = 'sm://snagsby/acceptance'
        if single_source in self.test_source:
            result = subprocess.run(
                [self.snagsby_path, single_source],
                capture_output=True,
                text=True
            )
            self.assertEqual(result.returncode, 0)
            # Should have keys from the single secret
            self.assertIn('TRICKY_CHARACTERS', result.stdout)
            self.assertIn('STARTS_WITH_HASH', result.stdout)

    def test_environment_variable_source(self):
        """Test SNAGSBY_SOURCE environment variable."""
        env = os.environ.copy()
        env['SNAGSBY_SOURCE'] = self.test_source
        result = subprocess.run(
            [self.snagsby_path],
            capture_output=True,
            text=True,
            env=env
        )
        self.assertEqual(result.returncode, 0)
        self.assertIn('export ', result.stdout)


if __name__ == '__main__':
    # Run tests with verbose output
    unittest.main(verbosity=2)
