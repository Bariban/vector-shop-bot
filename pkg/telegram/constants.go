package telegram

const (
	RndCmd       = "/rnd"
	HelpCmd      = "/help"
	StartCmd     = "/start"
	SalesCmd     = "/sales"
	ReplenishCmd = "/replenish"
	ContinueCmd  = "/continue"
)

const (
	EditProductNameCmd     = "edit_product_name"
	EditProductCountCmd    = "edit_product_count"
	EditProductPurchaseCmd = "edit_product_purchase"
	EditProductSellingCmd  = "edit_product_selling"
	EditProductCmd         = "edit_product"
	ConfirmEditProductCmd  = "confirm_edit_product"
	DelProductCmd          = "del_product"
	ConfirmDelProductCmd   = "confirm_del_product"
	ActionsProductCmd      = "actions_product"
	AddProductCmd          = "add_product"
	ListCmd                = "list"
	AddItemToCartCmd       = "add_item_to_cart"
	ReduceItemInCartCmd    = "reduce_item_in_cart"
	RemoveItemFromCartCmd  = "remove_item_from_cart"
	EditCountItemInCartCmd = "edit_count_item_in_cart"
	DiscountItemInCartCmd  = "discount_item_in_cart"
)

const (
	PayTypeCashCmd = "pay_type_cash"
	PayTypeKaspiCmd = "pay_type_kaspi"
)
const (
	stateWaitingForPhoto         = 1
	stateWaitingForName          = 2
	stateWaitingForDescription   = 3
	stateWaitingForCount         = 4
	stateWaitingForPurchasePrice = 5
	stateWaitingForSellingPrice  = 6
)

const (
	stateWaitingForEditName          = 7
	stateWaitingForEditCount         = 8
	stateWaitingForEditPurchasePrice = 9
	stateWaitingForEditSellingPrice  = 10
	stateIdle                        = 11
)

const (
	stateEditCountItemInCart = 12
	stateDiscountProductInCart = 13
)

var addProductStates = map[int]bool{
	stateWaitingForPhoto:         true,
	stateWaitingForName:          true,
	stateWaitingForDescription:   true,
	stateWaitingForCount:         true,
	stateWaitingForPurchasePrice: true,
	stateWaitingForSellingPrice:  true,
}

var editProductStates = map[int]bool{
	stateWaitingForEditName:          true,
	stateWaitingForEditCount:         true,
	stateWaitingForEditPurchasePrice: true,
	stateWaitingForEditSellingPrice:  true,
}

var makeCartStates = map[int]bool{
	stateEditCountItemInCart: true,
	stateDiscountProductInCart :true,
}

const (
	AddProductText  = "Добавить товар"
	SaleProductText = "Продажа"
	MenuText        = "Меню"
	PaymentText     = "Оплата"
)
