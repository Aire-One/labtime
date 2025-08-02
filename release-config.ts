export default {
  changeTypes: [
    {
      title: '💥 Breaking changes',
      labels: ['breaking'],
      bump: 'major',
      weight: 3,
    },
    {
      title: '🔒 Security',
      labels: ['security'],
      bump: 'patch',
      weight: 2,
    },
    {
      title: '✨ Features',
      labels: ['feature'],
      bump: 'minor',
      weight: 1,
    },
    {
      title: '📈 Enhancement',
      labels: ['enhancement', 'refactor'],
      bump: 'minor',
    },
    {
      title: ' Bug Fixes',
      labels: ['bug'],
      bump: 'patch',
    },
    {
      title: '📚 Documentation',
      labels: ['documentation'],
      bump: 'patch',
    },
    {
      title: '📦️ Dependency',
      labels: ['dependencies'],
      bump: 'patch',
      weight: -1,
    },
    {
      title: 'Misc',
      labels: ['chore'],
      bump: 'patch',
      default: true,
      weight: -2,
    },
  ],
  skipLabels: ['skip-changelog'],
};
