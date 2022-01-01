import { message } from 'antd';
import { MessageApi } from 'antd/lib/message';

const customMessageApi: MessageApi = message;

export { customMessageApi as message };
