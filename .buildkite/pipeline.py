import yaml

E2E_SNAGSBY_SOURCES = (
    (
        'Secrets Manager',
        'sm://snagsby/acceptance',
    ),
    (
        'SSM Params',
        'ssm://snagsby/acceptance',
    ),
)


def get_pipeline():
    steps = [
        {
            'command': 'make e2e',
            'label': 'e2e {}'.format(name),
            'agents': {
                'queue': 'dev',
            },
            'plugins': {
                'docker': {
                    'image': 'golang:1.12',
                    'workdir': '/go/src/github.com/roverdotcom/snagsby',
                    'environment': [
                        'AWS_REGION=us-west-2',
                        'SNAGSBY_E2E_SOURCE={}'.format(source),
                    ],
                },
            },
        }
        for name, source in E2E_SNAGSBY_SOURCES
    ]

    return {
        'steps': steps,
    }


if __name__ == "__main__":
    print(yaml.dump(get_pipeline()))
