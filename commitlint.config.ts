import type { UserConfig } from '@commitlint/types';

const Configuration: UserConfig = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'scope-empty': [2, 'never'],
    'scope-enum': [
      2,
      'always',
      [
        'api',
        'ui',
        // infra / cross-cutting
        'deps',
        'release',
        'tools',
        'deploy',
        'docs',
      ],
    ],
    'header-max-length': [2, 'always', 80],
    'body-max-line-length': [2, 'always', 80],
  },
};

export default Configuration;
