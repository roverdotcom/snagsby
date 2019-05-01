import yaml

E2E_SNAGSBY_SOURCES = [
    'sm://snagsby/acceptance',
    'ssm://snagsby/acceptance',
]


def get_pipeline():
    steps = [
        {
            'command': 'make e2e',
            'label': 'e2e {}'.format(source),
            'agents': {
                'queue': 'dev',
            },
            'plugins': {
                'docker': {
                    'image': 'golang:1.10',
                    'workdir': '/go/src/github.com/roverdotcom/snagsby',
                    'environment': [
                        'AWS_REGION=us-west-2',
                        'SNAGSBY_E2E_SOURCE={}'.format(source),
                    ],
                },
            },
        }
        for source in E2E_SNAGSBY_SOURCES
    ]

    return {
        'steps': steps,
    }


if __name__ == "__main__":
    print(yaml.dump(get_pipeline()))
