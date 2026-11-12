import BaseApi from './BaseApi';

class HistoryApi extends BaseApi {
  getList = (params) => this.http('/v1/history', params);

  getStats = (params) => this.http('/v1/history/stats', params);

  // data: the notification_settings object as it is in the form
  testNotify = (data) => this.http('/v1/notify/test', data, 'POST');
}
export default new HistoryApi();
