export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      ['feat', 'fix', 'docs', 'style', 'refactor', 'perf', 'test', 'build', 'ci', 'chore', 'revert'],
    ],
    'scope-enum': [2, 'always', ['server', 'dash', 'get-opus', 'ci', 'deps']],
    'header-max-length': [2, 'always', 70],
  },
};
