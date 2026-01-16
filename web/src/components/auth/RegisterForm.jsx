/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { getLogo, getSystemName } from '../../helpers';
import { Button, Card } from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const RegisterForm = () => {
  const { t } = useTranslation();
  const logo = getLogo();
  const systemName = getSystemName();

  // Save invitation code to localStorage
  useEffect(() => {
    const affCode = new URLSearchParams(window.location.search).get('aff');
    if (affCode) {
      localStorage.setItem('aff', affCode);
    }
  }, []);

  return (
    <div className='relative overflow-hidden bg-gray-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8'>
      {/* 背景模糊晕染球 */}
      <div
        className='blur-ball blur-ball-indigo'
        style={{ top: '-80px', right: '-80px', transform: 'none' }}
      />
      <div
        className='blur-ball blur-ball-teal'
        style={{ top: '50%', left: '-120px' }}
      />
      <div className='w-full max-w-sm mt-[60px]'>
        <div className='flex flex-col items-center'>
          <div className='w-full max-w-md'>
            <div className='flex items-center justify-center mb-6 gap-2'>
              <img src={logo} alt='Logo' className='h-10 rounded-full' />
              <Title heading={3} className='!text-gray-800'>
                {systemName}
              </Title>
            </div>

            <Card className='border-0 !rounded-2xl overflow-hidden'>
              <div className='flex justify-center pt-6 pb-2'>
                <Title heading={3} className='text-gray-800 dark:text-gray-200'>
                  {t('注册')}
                </Title>
              </div>
              <div className='px-6 py-8'>
                <div className='text-center mb-6'>
                  <div className='w-16 h-16 mx-auto mb-4 bg-blue-100 dark:bg-blue-900 rounded-full flex items-center justify-center'>
                    <IconKey size='extra-large' className='text-blue-600 dark:text-blue-400' />
                  </div>
                  <Text type='secondary' className='block mb-2'>
                    {t('本系统仅支持 ZKP 登录')}
                  </Text>
                  <Text type='tertiary' size='small' className='block mb-3'>
                    {t('请使用您的 ZKP 凭证进行登录，首次登录将自动创建账户')}
                  </Text>
                  <div className='mt-4 p-3 bg-blue-50 dark:bg-blue-900/30 rounded-lg'>
                    <Text type='tertiary' size='small' className='block mb-2'>
                      {t('还没有 ZKP 凭证？')}
                    </Text>
                    <Text type='tertiary' size='small' className='block'>
                      {t('前往')}{' '}
                      <a
                        href='https://ai.web3.club'
                        target='_blank'
                        rel='noopener noreferrer'
                        className='text-blue-600 hover:text-blue-800 font-medium underline'
                      >
                        ai.web3.club
                      </a>{' '}
                      {t('购买会员生成密钥即可登录')}
                    </Text>
                  </div>
                  <div className='mt-3 p-3 bg-green-50 dark:bg-green-900/30 rounded-lg'>
                    <Text type='success' size='small' className='block'>
                      ✨ {t('注册送100美元额度，每天签到领取1美元')}
                    </Text>
                  </div>
                </div>

                <Link to='/login'>
                  <Button
                    theme='solid'
                    type='primary'
                    className='w-full h-12 flex items-center justify-center bg-black text-white !rounded-full hover:bg-gray-800 transition-colors'
                    icon={<IconKey size='large' />}
                  >
                    <span className='ml-2'>{t('前往 ZKP 登录')}</span>
                  </Button>
                </Link>

                <div className='mt-6 text-center'>
                  <Text size='small' type='tertiary'>
                    {t('已有账户？')}{' '}
                    <Link
                      to='/login'
                      className='text-blue-600 hover:text-blue-800 font-medium'
                    >
                      {t('立即登录')}
                    </Link>
                  </Text>
                </div>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
};

export default RegisterForm;
