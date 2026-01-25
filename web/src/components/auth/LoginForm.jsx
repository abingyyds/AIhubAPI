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

import React, { useContext, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User';
import {
  API,
  getLogo,
  showError,
  showSuccess,
  updateAPI,
  getSystemName,
  setUserData,
} from '../../helpers';
import { Button, Card, Form, Checkbox } from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const LoginForm = () => {
  let navigate = useNavigate();
  const { t } = useTranslation();
  const [zkpCode, setZkpCode] = useState('');
  const [searchParams] = useSearchParams();
  const [, userDispatch] = useContext(UserContext);
  const [loginLoading, setLoginLoading] = useState(false);
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [hasUserAgreement, setHasUserAgreement] = useState(false);
  const [hasPrivacyPolicy, setHasPrivacyPolicy] = useState(false);

  const logo = getLogo();
  const systemName = getSystemName();

  let affCode = new URLSearchParams(window.location.search).get('aff');
  if (affCode) {
    localStorage.setItem('aff', affCode);
  }

  const [status] = useState(() => {
    const savedStatus = localStorage.getItem('status');
    return savedStatus ? JSON.parse(savedStatus) : {};
  });

  useEffect(() => {
    setHasUserAgreement(status.user_agreement_enabled || false);
    setHasPrivacyPolicy(status.privacy_policy_enabled || false);
  }, [status]);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('未登录或登录已过期，请重新登录'));
    }
    
    const zkpParam = searchParams.get('zkp');
    if (zkpParam) {
      try {
        const decodedZkp = atob(zkpParam);
        setZkpCode(decodedZkp);
        setAgreedToTerms(true);
        handleZkpLogin(decodedZkp);
      } catch (error) {
        console.error('Auto login error:', error);
      }
    }
  }, [searchParams, t]);

  async function handleZkpLogin(zkpCodeOverride) {
    const isAutoLogin = typeof zkpCodeOverride === 'string';
    const finalZkpCode = isAutoLogin ? zkpCodeOverride : zkpCode;

    if (!isAutoLogin && (hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showError(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }

    if (!finalZkpCode.trim()) {
      showError(t('请输入 ZKP Code'));
      return;
    }

    setLoginLoading(true);
    try {
      // Get invitation code from localStorage
      const affCode = localStorage.getItem('aff');

      const res = await API.post('/api/oauth/zkp', {
        zkpCode: finalZkpCode.trim(),
        affCode: affCode || undefined,
      });
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        showSuccess(t('登录成功！'));
        navigate('/console');
      } else {
        // Handle specific error codes
        switch (message) {
          case 'INVALID_PAYLOAD':
            showError(t('请求格式错误'));
            break;
          case 'INVALID_ZKP_CODE':
            showError(t('ZKP Code 格式无效'));
            break;
          case 'PROOF_INVALID':
            showError(t('ZKP 验证失败，请检查您的凭证'));
            break;
          case 'NOT_CLUB_MEMBER':
            showError(t('您不是 AI Club 成员，无法登录'));
            break;
          default:
            showError(message || t('登录失败'));
        }
      }
    } catch (error) {
      console.error('ZKP login error:', error);
      showError(t('登录失败，请重试'));
    } finally {
      setLoginLoading(false);
    }
  }

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
                  {t('ZKP 登录')}
                </Title>
              </div>
              <div className='px-2 py-8'>
                <div className='mb-4 text-center'>
                  <Text type='secondary'>
                    {t('使用您的 ZKP 凭证登录 AI Club')}
                  </Text>
                </div>

                <Form className='space-y-4'>
                  <Form.TextArea
                    field='zkpCode'
                    label={t('ZKP Code')}
                    placeholder={t('请输入您的 ZKP Code（9个数字，用逗号分隔）')}
                    value={zkpCode}
                    onChange={setZkpCode}
                    autosize={{ minRows: 3, maxRows: 6 }}
                    prefix={<IconKey />}
                  />

                  {(hasUserAgreement || hasPrivacyPolicy) && (
                    <div className='pt-2'>
                      <Checkbox
                        checked={agreedToTerms}
                        onChange={(e) => setAgreedToTerms(e.target.checked)}
                      >
                        <Text size='small' className='text-gray-600'>
                          {t('我已阅读并同意')}
                          {hasUserAgreement && (
                            <a
                              href='/user-agreement'
                              target='_blank'
                              rel='noopener noreferrer'
                              className='text-blue-600 hover:text-blue-800 mx-1'
                            >
                              {t('用户协议')}
                            </a>
                          )}
                          {hasUserAgreement && hasPrivacyPolicy && t('和')}
                          {hasPrivacyPolicy && (
                            <a
                              href='/privacy-policy'
                              target='_blank'
                              rel='noopener noreferrer'
                              className='text-blue-600 hover:text-blue-800 mx-1'
                            >
                              {t('隐私政策')}
                            </a>
                          )}
                        </Text>
                      </Checkbox>
                    </div>
                  )}

                  <div className='pt-4'>
                    <Button
                      theme='solid'
                      type='primary'
                      className='w-full h-12 flex items-center justify-center bg-black text-white !rounded-full hover:bg-gray-800 transition-colors'
                      icon={<IconKey size='large' />}
                      onClick={handleZkpLogin}
                      loading={loginLoading}
                      disabled={
                        (hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms
                      }
                    >
                      <span className='ml-2'>{t('使用 ZKP 登录')}</span>
                    </Button>
                  </div>

                  <div className='mt-4 text-center'>
                    <Text size='small' type='tertiary'>
                      {t('还没有 ZKP Code？')}{' '}
                      <a
                        href='https://ai.web3.club'
                        target='_blank'
                        rel='noopener noreferrer'
                        className='text-blue-600 hover:text-blue-800 font-medium'
                      >
                        {t('前往 ai.web3.club 购买会员生成')}
                      </a>
                    </Text>
                  </div>
                </Form>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginForm;
