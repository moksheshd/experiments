package com.mokshesh;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Map.Entry;

interface IOrder {
    void setName(String name);
    String getName();
    void setPrice(int price);
    int getPrice();
}
interface IOrderSystem {
    void addToCart(IOrder order);
    void removeFromCart(IOrder order);
    int calculateTotalAmount();
    Map<String, Integer> categoryDiscounts();
    Map<String, Integer> cartItems();
}

class Order implements IOrder {
    String name;
    int price;

    public void setName(String name){
        this.name = name;
    }

    public String getName(){
        return this.name;
    }

    public void setPrice(int price){
        this.price = price;
    }

    public int getPrice(){
        return price;

    }
}

class CartItem {
    int price;
    int quantity;
    CartItem(int price, int quantity){
        this.price = price;
        this.quantity = quantity;
    }
}

class OrderSystem implements IOrderSystem {
    Map<String, CartItem> cart = new LinkedHashMap<>();
    Map<String, Integer> categoryDiscountMap = new HashMap<>();

    public void addToCart(IOrder order) {
        CartItem item = cart.get(order.getName());
        if(item == null){
            cart.put(order.getName(), new CartItem(order.getPrice(), 1));
        } else {
            item.quantity++;
        }
    }
    public void removeFromCart(IOrder order){
        CartItem item = cart.get(order.getName());
        if(item != null){
            item.quantity--;
            if(item.quantity == 0) {
                cart.remove(order.getName());
            }
        }
    }
    public int calculateTotalAmount() {
        int total = 0;
        for(Entry<String, CartItem> details : cart.entrySet()) {
            CartItem item = details.getValue();
            int subTotal = item.price * item.quantity;
            int discount = 1;
            int finalDiscount = 0;
            if(item.price > 10) {
                discount = 10;
            } else if(item.price > 10 && item.price <= 20) {
                discount = 20;
            } else if(item.price > 20) {
                discount = 30;
            }
            finalDiscount = ((subTotal * discount)/100);
            if(discount == 10){                
                categoryDiscountMap.put("Cheap", categoryDiscountMap.getOrDefault("Cheap", 0) + finalDiscount);
            } else if(discount == 20){                
                categoryDiscountMap.put("Moderate", categoryDiscountMap.getOrDefault("Moderate", 0) + finalDiscount);
            } else if(discount == 30){                
                categoryDiscountMap.put("Expensive", categoryDiscountMap.getOrDefault("Expensive", 0) + finalDiscount);
            }
            subTotal = subTotal - finalDiscount;
            total = total + subTotal;
        }
        return total;
    }
    public Map<String, Integer> categoryDiscounts(){
        return categoryDiscountMap;
    }
    public Map<String, Integer> cartItems() {        
        Map<String, Integer> cartItemCount = new LinkedHashMap<String, Integer>();
        cart.forEach((k, item) -> cartItemCount.put(k, item.quantity));
        return cartItemCount;
    }
}

public class OrderManagement {
    public static void main(String[] args) throws NumberFormatException, IOException {
        BufferedReader br = new BufferedReader(new InputStreamReader(System.in));
        PrintWriter textWriter = new PrintWriter(System.out);
        IOrderSystem orderSystem = new OrderSystem();
        int oCount = Integer.parseInt(br.readLine().trim());
        for (int i = 1; i <= oCount; i++) {
            String[] a = br.readLine().trim().split(" ");
            IOrder e = new Order();
            e.setName("Order-" + a[0]);
            e.setPrice(Integer.parseInt(a[1]));
            orderSystem.addToCart(e);
        }
        int totalAmount = orderSystem.calculateTotalAmount();
        textWriter.println("Total Amount: " + totalAmount);
        Map<String, Integer> categoryDiscounts = orderSystem.categoryDiscounts();
        for (Map.Entry<String, Integer> entry : categoryDiscounts.entrySet()) {
            if(entry.getValue() > 0) {
                textWriter.println(entry.getKey() + " Category Discount: " + entry.getValue());
            }
        }
        Map<String, Integer> cartItems = orderSystem.cartItems();
        for (Map.Entry<String, Integer> entry : cartItems.entrySet()) {
            textWriter.println(entry.getKey() + " (" + entry.getValue() + " items)");
        }
        textWriter.flush();
        textWriter.close();        
    }

}
