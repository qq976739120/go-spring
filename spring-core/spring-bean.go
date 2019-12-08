/*
 * Copyright 2012-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package SpringCore

import (
	"reflect"
	"strconv"
	"strings"
)

var (
	_VALID_BEAN_KINDS = []reflect.Kind{ // 哪些类型可以成为 Bean.
		reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func,
	}

	_VALID_RECEIVER_KINDS = []reflect.Kind{ // 哪些类型可以成为 Bean 的接收者
		reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func,
	}
)

//
// 定义 SpringBean 接口
//
type SpringBean interface {
	Bean() interface{}
	Type() reflect.Type
	Value() reflect.Value
	TypeName() string
}

//
// 保存原始对象的 SpringBean
//
type OriginalBean struct {
	bean     interface{}
	rType    reflect.Type  // 类型
	typeName string        // 原始类型的全限定名
	rValue   reflect.Value // 值
}

//
// 工厂函数
//
func NewOriginalBean(bean interface{}) *OriginalBean {

	t, ok := IsValidBean(bean)
	if !ok {
		panic("bean must be ptr or slice or map or func")
	}

	return &OriginalBean{
		bean:     bean,
		rType:    t,
		typeName: TypeName(t),
		rValue:   reflect.ValueOf(bean),
	}
}

func (b *OriginalBean) Bean() interface{} {
	return b.bean
}

func (b *OriginalBean) Type() reflect.Type {
	return b.rType
}

func (b *OriginalBean) Value() reflect.Value {
	return b.rValue
}

func (b *OriginalBean) TypeName() string {
	return b.typeName
}

//
// 保存构造函数的 SpringBean
//
type ConstructorBean struct {
	OriginalBean

	fn      interface{}
	fnType  reflect.Type
	fnValue reflect.Value
	tags    []string
}

//
// 工厂函数，所有 tag 都必须同时有或者同时没有序号。
//
func NewConstructorBean(fn interface{}, tags ...string) *ConstructorBean {

	fnType := reflect.TypeOf(fn)

	if fnType.Kind() != reflect.Func {
		panic("constructor must be func")
	}

	if fnType.NumOut() != 1 {
		panic("constructor must be one out")
	}

	fnTags := make([]string, fnType.NumIn())

	if len(tags) > 0 {
		indexed := false // 是否包含序号

		if tag := tags[0]; tag != "" {
			if i := strings.Index(tag, ":"); i > 0 {
				_, err := strconv.Atoi(tag[:i])
				indexed = err == nil
			}
		}

		if indexed { // 有序号
			for _, tag := range tags {
				index := strings.Index(tag, ":")
				if index <= 0 {
					panic("tag \"" + tag + "\" should have index")
				}
				i, err := strconv.Atoi(tag[:index])
				if err != nil {
					panic("tag \"" + tag + "\" should have index")
				}
				fnTags[i] = tag[index+1:]
			}

		} else { // 无序号
			for i, tag := range tags {
				if index := strings.Index(tag, ":"); index > 0 {
					_, err := strconv.Atoi(tag[:index])
					if err == nil {
						panic("tag \"" + tag + "\" should no index")
					}
				}
				fnTags[i] = tag
			}
		}
	}

	t := fnType.Out(0)
	k := t.Kind()
	v := reflect.New(t)

	for i := range _VALID_BEAN_KINDS {
		if _VALID_BEAN_KINDS[i] == k {
			v = v.Elem()
			break
		}
	}

	// 重新确定类型
	t = v.Type()

	return &ConstructorBean{
		OriginalBean: OriginalBean{
			bean:     v.Interface(),
			rType:    t,
			typeName: TypeName(t),
			rValue:   v,
		},
		fn:      fn,
		fnType:  fnType,
		fnValue: reflect.ValueOf(fn),
		tags:    fnTags,
	}
}

//
// 定义 Bean 的状态值
//
type BeanStatus int

const (
	BeanStatus_Default  = BeanStatus(0) // 默认状态
	BeanStatus_Resolved = BeanStatus(1) // 已决议状态
	BeanStatus_Wiring   = BeanStatus(2) // 正在绑定状态
	BeanStatus_Wired    = BeanStatus(3) // 绑定完成状态
)

//
// 定义 BeanDefinition 类型
//
type BeanDefinition struct {
	SpringBean

	Name      string     // 名称
	status    BeanStatus // 状态
	cond      Condition  // 注册条件
	profile   string     // 运行环境
	dependsOn []string   // 非直接依赖
	primary   bool       // 主版本
}

//
// 工厂函数
//
func NewBeanDefinition(bean SpringBean, name string) *BeanDefinition {

	// 生成默认名称
	if name == "" {
		name = bean.Type().String()
	}

	return &BeanDefinition{
		SpringBean: bean,
		Name:       name,
		status:     BeanStatus_Default,
	}
}

//
// 测试类型全限定名和 Bean 名称是否都能匹配。
//
func (d *BeanDefinition) Match(typeName string, beanName string) bool {

	typeIsSame := false
	if typeName == "" || d.TypeName() == typeName {
		typeIsSame = true
	}

	nameIsSame := false
	if beanName == "" || d.Name == beanName {
		nameIsSame = true
	}

	return typeIsSame && nameIsSame
}

//
// 将 Bean 转换为 BeanDefinition 对象
//
func ToBeanDefinition(name string, i interface{}) *BeanDefinition {
	bean := NewOriginalBean(i)
	return NewBeanDefinition(bean, name)
}

//
// 将构造函数转换为 BeanDefinition 对象
//
func FnToBeanDefinition(name string, fn interface{}, tags ...string) *BeanDefinition {
	bean := NewConstructorBean(fn, tags...)
	return NewBeanDefinition(bean, name)
}
