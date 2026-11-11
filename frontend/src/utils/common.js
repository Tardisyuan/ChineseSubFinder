export const deepCopy = (obj) => JSON.parse(JSON.stringify(obj));

export const gotoGithubIssuePage = () => {
  window.open(`https://github.com/Tardisyuan/ChineseSubFinder/issues/new`, '_blank');
};

export const isImdbId = (str) => {
  if (!str) return false;
  if (str === 'tt00000') return false;
  return /^tt\d{7,8}$/.test(str);
};
